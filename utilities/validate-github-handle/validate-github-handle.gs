var CHUNK_SIZE = 8000;
var indexMemo_ = null;
var API = 'https://api.github.com';
var GHE_LEVELS = ['graduated', 'incubating'];

function token_() {
  var t = PropertiesService.getScriptProperties().getProperty('GITHUB_TOKEN');
  if (!t) throw new Error('Set Script Property GITHUB_TOKEN (needs read:org)');
  return t;
}

function headers_(auth) {
  var h = { Accept: 'application/vnd.github+json', 'User-Agent': 'cncf-handle-validator' };
  if (auth) h.Authorization = 'Bearer ' + token_();
  return h;
}

function fetchText_(url, auth) {
  Utilities.sleep(300);
  var r = UrlFetchApp.fetch(url, { headers: headers_(auth), muteHttpExceptions: true });
  return r.getResponseCode() === 200 ? r.getContentText() : null;
}

function fetchAllThrottled_(requests) {
  var out = [];
  var size = 10;
  for (var i = 0; i < requests.length; i += size) {
    if (i > 0) Utilities.sleep(1500);
    out = out.concat(UrlFetchApp.fetchAll(requests.slice(i, i + size)));
  }
  return out;
}

function landscapeProjects_() {
  var text = fetchText_('https://raw.githubusercontent.com/cncf/landscape/master/landscape.yml');
  if (!text) throw new Error('Cannot fetch landscape.yml');
  var out = [];
  var cur = null;
  text.split('\n').forEach(function (line) {
    if (/^\s*- item:/.test(line)) {
      if (cur && cur.level && cur.org) out.push(cur);
      cur = { level: '', org: '', name: '' };
      return;
    }
    if (!cur) return;
    var m;
    if ((m = line.match(/^\s+name:\s*(.+?)\s*$/)) && !cur.name) cur.name = m[1].replace(/^["']|["']$/g, '');
    else if ((m = line.match(/^\s+project:\s*(\w+)\s*$/))) cur.level = m[1];
    else if ((m = line.match(/^\s+repo_url:\s*https:\/\/github\.com\/([^\/\s]+)/))) cur.org = m[1];
  });
  if (cur && cur.level && cur.org) out.push(cur);
  return out;
}

function parseMaintainers_(yaml) {
  var handles = [];
  var team = null;
  var inMembers = false;
  function flush() {
    if (team && !team.unmanaged) handles = handles.concat(team.members);
    team = null;
  }
  yaml.split('\n').forEach(function (line) {
    var m;
    if (/^\s*-?\s*name:/.test(line) && /^\s{4,}-\s*name:|^\s{6,}-\s*name:/.test(line)) {
      flush();
      team = { members: [], unmanaged: false };
      inMembers = false;
    } else if (/^\s*managed:\s*false/.test(line) && team) {
      team.unmanaged = true;
    } else if (/^\s*members:\s*$/.test(line)) {
      inMembers = true;
    } else if (/^\s*\w+:/.test(line)) {
      inMembers = false;
    } else if (inMembers && team && (m = line.match(/^\s*-\s*@?([A-Za-z0-9][A-Za-z0-9-]*)\s*(#.*)?$/))) {
      team.members.push(m[1]);
    }
  });
  flush();
  return handles;
}

function dotProjectRows_(projects) {
  var seenOrgs = {};
  var reqs = [];
  projects.forEach(function (p) {
    if (seenOrgs[p.org]) return;
    seenOrgs[p.org] = true;
    reqs.push({
      org: p.org,
      name: p.name,
      request: {
        url: 'https://raw.githubusercontent.com/' + p.org + '/.project/HEAD/maintainers.yaml',
        muteHttpExceptions: true,
      },
    });
  });
  var resps = fetchAllThrottled_(reqs.map(function (r) { return r.request; }));
  var rows = [];
  resps.forEach(function (resp, i) {
    var org = reqs[i].org;
    var yamls = [];
    if (resp.getResponseCode() === 200) {
      yamls.push({ project: reqs[i].name, text: resp.getContentText() });
    } else {
      var listing = fetchText_(API + '/repos/' + org + '/.project/contents/', true);
      if (!listing) return;
      JSON.parse(listing).forEach(function (e) {
        if (e.type !== 'dir' || e.name.charAt(0) === '.') return;
        var t = fetchText_('https://raw.githubusercontent.com/' + org + '/.project/HEAD/' + e.name + '/maintainers.yaml');
        if (t) yamls.push({ project: e.name, text: t });
      });
    }
    yamls.forEach(function (y) {
      parseMaintainers_(y.text).forEach(function (h) {
        rows.push([h.toLowerCase(), 'dot-project', y.project]);
      });
    });
  });
  return rows;
}

function k8sRows_() {
  var tree = fetchText_(API + '/repos/kubernetes/org/git/trees/main?recursive=1', true);
  if (!tree) throw new Error('Cannot read kubernetes/org tree');
  var files = JSON.parse(tree).tree.filter(function (e) {
    return /^config\/[^\/]+\/org\.yaml$/.test(e.path);
  });
  var resps = fetchAllThrottled_(files.map(function (f) {
    return { url: 'https://raw.githubusercontent.com/kubernetes/org/main/' + f.path, muteHttpExceptions: true };
  }));
  var rows = [];
  resps.forEach(function (resp, i) {
    if (resp.getResponseCode() !== 200) return;
    var org = files[i].path.split('/')[1];
    var key = null;
    resp.getContentText().split('\n').forEach(function (line) {
      var m;
      if ((m = line.match(/^(\w+):\s*$/))) key = m[1];
      else if (/^\S/.test(line) && line.charAt(0) !== '-') key = null;
      else if ((key === 'members' || key === 'admins') && (m = line.match(/^-\s*@?([A-Za-z0-9][A-Za-z0-9-]*)\s*$/))) {
        rows.push([m[1].toLowerCase(), 'k8s-org', org + (key === 'admins' ? ' (admin)' : '')]);
      }
    });
  });
  return rows;
}

var TOC_RAW = 'https://raw.githubusercontent.com/cncf/toc/main/';

function ownersHandles_(text) {
  var out = [];
  var key = null;
  text.split('\n').forEach(function (line) {
    var m;
    if (/^\s*#/.test(line)) return;
    if ((m = line.match(/^(\w+):/))) key = m[1];
    else if ((key === 'approvers' || key === 'reviewers') && (m = line.match(/^\s*-\s*@?([A-Za-z0-9][A-Za-z0-9-]*)\s*(#.*)?$/))) {
      out.push([m[1].toLowerCase(), key]);
    }
  });
  return out;
}

function tocAndTagRows_() {
  var rows = [];
  var root = fetchText_(TOC_RAW + 'OWNERS');
  if (!root) throw new Error('Cannot fetch cncf/toc OWNERS');
  ownersHandles_(root).forEach(function (h) {
    rows.push([h[0], 'toc', 'TOC ' + h[1]]);
  });
  var tags = fetchText_(TOC_RAW + 'tags.yaml');
  if (!tags) throw new Error('Cannot fetch cncf/toc tags.yaml');
  var dirs = [];
  var names = {};
  var tag = '';
  var section = null;
  var labels = { chairs: 'chair', tech_leads: 'tech lead', toc_liaison: 'TOC liaison' };
  tags.split('\n').forEach(function (line) {
    var m;
    if (/^\s*#/.test(line)) return;
    if ((m = line.match(/^ {2}- dir:\s*(\S+)/))) {
      dirs.push(m[1]);
      tag = m[1];
    } else if ((m = line.match(/^ {4}name:\s*(.+?)\s*$/))) {
      names[tag] = m[1];
    } else if ((m = line.match(/^\s+(\w+):/)) && !/^(github|lfx_id|name|company|email|term|start|end)$/.test(m[1])) {
      section = labels[m[1]] ? m[1] : null;
    }
    if (section && (m = line.match(/^\s*-\s*github:\s*@?([A-Za-z0-9][A-Za-z0-9-]*)/))) {
      rows.push([m[1].toLowerCase(), 'tag', (names[tag] || tag) + ' ' + labels[section]]);
    }
  });
  var resps = fetchAllThrottled_(dirs.map(function (d) {
    return { url: TOC_RAW + 'tags/' + d + '/OWNERS', muteHttpExceptions: true };
  }));
  resps.forEach(function (r, i) {
    if (r.getResponseCode() !== 200) return;
    ownersHandles_(r.getContentText()).forEach(function (h) {
      rows.push([h[0], 'tag', (names[dirs[i]] || dirs[i]) + ' ' + h[1]]);
    });
  });
  return rows;
}

function staffRows_() {
  var text = fetchText_('https://raw.githubusercontent.com/cncf/people/refs/heads/main/people.json');
  if (!text) throw new Error('Cannot fetch cncf/people people.json');
  var rows = [];
  JSON.parse(text).forEach(function (p) {
    if (!p.category || p.category.indexOf('Staff') < 0 || !p.github) return;
    var m = String(p.github).match(/github\.com\/([A-Za-z0-9][A-Za-z0-9-]*)/i) || String(p.github).match(/^@?([A-Za-z0-9][A-Za-z0-9-]*)$/);
    if (m) rows.push([m[1].toLowerCase(), 'staff', 'CNCF staff']);
  });
  return rows;
}

function saveIndex_(index, orgs) {
  var props = PropertiesService.getScriptProperties();
  var all = props.getKeys();
  all.forEach(function (k) {
    if (/^IDX_\d+$/.test(k)) props.deleteProperty(k);
  });
  var blob = Utilities.gzip(Utilities.newBlob(JSON.stringify(index), 'application/json'));
  var b64 = Utilities.base64Encode(blob.getBytes());
  var chunks = Math.ceil(b64.length / CHUNK_SIZE);
  var out = { IDX_COUNT: String(chunks), GHE_ORGS: orgs.join(','), UPDATED: new Date().toISOString() };
  for (var i = 0; i < chunks; i++) out['IDX_' + i] = b64.substr(i * CHUNK_SIZE, CHUNK_SIZE);
  props.setProperties(out);
  indexMemo_ = null;
}

function loadIndex_() {
  if (indexMemo_) return indexMemo_;
  var props = PropertiesService.getScriptProperties().getProperties();
  var n = parseInt(props.IDX_COUNT, 10);
  if (!n) throw new Error('Index is empty: run refreshIndex() in the library project');
  var b64 = '';
  for (var i = 0; i < n; i++) b64 += props['IDX_' + i] || '';
  var bytes = Utilities.base64Decode(b64);
  var json = Utilities.ungzip(Utilities.newBlob(bytes, 'application/x-gzip')).getDataAsString();
  indexMemo_ = { map: JSON.parse(json), orgs: props.GHE_ORGS ? props.GHE_ORGS.split(',') : [] };
  return indexMemo_;
}

function refreshIndex() {
  var projects = landscapeProjects_();
  var rows = dotProjectRows_(projects).concat(k8sRows_()).concat(tocAndTagRows_()).concat(staffRows_());
  var orgs = [];
  var seen = {};
  projects.forEach(function (p) {
    if (GHE_LEVELS.indexOf(p.level) >= 0 && !seen[p.org]) {
      seen[p.org] = true;
      orgs.push(p.org);
    }
  });
  var index = {};
  rows.forEach(function (r) {
    var list = index[r[0]] || (index[r[0]] = []);
    if (!list.some(function (e) { return e[0] === r[1] && e[1] === r[2]; })) list.push([r[1], r[2]]);
  });
  saveIndex_(index, orgs);
}

function gheOrgsFor_(handle, orgs) {
  var cache = CacheService.getScriptCache();
  var key = 'ghe:' + handle;
  var hit = cache.get(key);
  if (hit !== null) return hit ? hit.split(',') : [];
  var slug = PropertiesService.getScriptProperties().getProperty('ENTERPRISE_SLUG');
  if (!slug) throw new Error('Set Script Property ENTERPRISE_SLUG');
  var query =
    'query($slug:String!,$q:String!,$orgs:[String!]){enterprise(slug:$slug){' +
    'members(first:20,query:$q,organizationLogins:$orgs){nodes{' +
    '... on EnterpriseUserAccount{login organizations(first:100){nodes{login}}}' +
    '... on User{login organizations(first:100){nodes{login}}}}}}}';
  var r = UrlFetchApp.fetch(API + '/graphql', {
    method: 'post',
    contentType: 'application/json',
    headers: headers_(true),
    muteHttpExceptions: true,
    payload: JSON.stringify({ query: query, variables: { slug: slug, q: handle, orgs: orgs } }),
  });
  var body = JSON.parse(r.getContentText());
  if (body.errors && !body.data) throw new Error(body.errors[0].message);
  var found = [];
  var ent = body.data && body.data.enterprise;
  ((ent && ent.members.nodes) || []).forEach(function (n) {
    if (!n || !n.login || n.login.toLowerCase() !== handle) return;
    ((n.organizations && n.organizations.nodes) || []).forEach(function (o) {
      if (orgs.indexOf(o.login) >= 0 && found.indexOf(o.login) < 0) found.push(o.login);
    });
    if (!found.length) found.push('enterprise');
  });
  cache.put(key, found.join(','), 21600);
  return found;
}

function normalizeHandle_(input) {
  var v = String(input || '').trim();
  v = v.replace(/^(https?:\/\/)?(www\.)?github\.com\//i, '').replace(/^@/, '');
  v = v.split(/[\/?#]/)[0];
  return /^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$/.test(v) ? v.toLowerCase() : '';
}

/**
 * Validates a GitHub handle against dot-project maintainers, GitHub
 * Enterprise graduated/incubating orgs, and the Kubernetes org config.
 * @param {string} handle GitHub handle (with or without @)
 * @return {string} Summary of matches
 * @customfunction
 */
function VALIDATE_HANDLE(handle) {
  if (Array.isArray(handle)) return handle.map(function (r) { return [VALIDATE_HANDLE(r[0])]; });
  var h = normalizeHandle_(handle);
  if (!h) return '';
  var idx = loadIndex_();
  var dot = [];
  var k8s = [];
  var toc = [];
  var tag = [];
  var staff = [];
  (idx.map[h] || []).forEach(function (r) {
    if (r[0] === 'dot-project') dot.push(r[1]);
    else if (r[0] === 'k8s-org') k8s.push(r[1]);
    else if (r[0] === 'toc') toc.push(r[1]);
    else if (r[0] === 'tag') tag.push(r[1]);
    else if (r[0] === 'staff') staff.push(r[1]);
  });
  var orgs = idx.orgs;
  var ghe = gheOrgsFor_(h, orgs);
  var parts = [];
  if (dot.length) parts.push('Maintainer: ' + dot.join(', '));
  if (ghe.length) parts.push('GHE: ' + ghe.join(', '));
  if (k8s.length) parts.push('K8s org');
  if (toc.length) parts.push('TOC: ' + toc.join(', '));
  if (tag.length) parts.push('TAG: ' + tag.join(', '));
  if (staff.length) parts.push('Staff');
  return parts.length ? 'VALID | ' + parts.join(' | ') : 'NOT FOUND';
}
