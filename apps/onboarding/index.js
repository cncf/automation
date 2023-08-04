import * as dotenv from 'dotenv'
dotenv.config()

import { writeFile } from 'fs/promises';
import * as fossa from  '../../libs/fossa/index.js';
import * as projects from  '../../libs/projects/src/index.js';

import { get } from 'https';
const githubToken = process.env.GITHUB_TOKEN;
const owner = 'cncf'; 
const repo = 'toc'; 
const labels = 'static-code-checks'; // filter issues for onboarding projects.  
const state = 'open'; 
const limit = 1000;

const onboardingIssues = `https://api.github.com/repos/${owner}/${repo}/issues?labels=${labels}&state=${state}&per_page=${limit}`;

const headers = {
  "User-Agent": "RobertKielty", // TODO xfer over to cncf-automation-bot
  Authorization: `Bearer ${githubToken}`,
  Accept: 'application/vnd.github+json',
};

let communityMaintainers = await projects.getCommunityMaintainerMap(process.env.MAINTAINER_SPREADSHEET_ID)
console.log(`communityMaintainers has ${communityMaintainers.size-1} projects`)

get(onboardingIssues, { headers }, response => {
  let data = '';
  response.on('data', chunk => {
    data += chunk;
  });

  response.on('end', () => {
    const issues = JSON.parse(data).map(issue => {
      return {
        number: issue.number,
        title: issue.title,
        labels: issue.labels.map(label => label.name),
        url: issue.url,
        body: issue.body
      };
    });

    // The cncf/toc repo issues track onboarding of new community projects
    issues.forEach(issue => {
        const proposedProjectName = getProjectName(issue.title)

        let projectMaintainersEmailAddrs = communityMaintainers.get(proposedProjectName)

        if (projectMaintainersEmailAddrs.length > 0) {
          fossa.getFossaTeamId(proposedProjectName).then(
            (fossaTeamId) => {
              if (fossaTeamId === null) {
                console.log(`Proposed project ${proposedProjectName} team not present in FOSSA, will create...`)
                fossa.createFossaTeam(proposedProjectName).then(
                  (teamResponse) => {
                    storeEvent("FOSSA_EVENT", "TEAM_CREATED", teamResponse)
                    addLabelToIssue(issue.number,"fossa-team-created")
                  }
                )
              } else {
                console.log(`FOSSA already has a ${proposedProjectName} team with ${fossaTeamId} ID`)
              }
              // fossa-team-populated label to issue ??

              // fossaTeamMembersIds = fossaGetTeamMemebers(fossaTeamId)
              // Iterate over the projects maintainer email addrs
              projectMaintainersEmailAddrs.forEach(maintainerEmail => { 
                var fossaUserIds = []
                var emailsNotRegisteredInFossa = [] 
                
                fossa.getFossaUserByEmail(maintainerEmail).then(fossaUserId => {
                  if (fossaUserId != null && !fossa.isExistingTeamMember(fossaUserId,fossaTeamId)) {
                    fossaUserIds.push(fossaUserId)
                  } else {
                    emailsNotRegisteredInFossa.push(maintainerEmail)
                  }
                })
                console.log(`fossaUserIds : ${fossaUserIds}`)
                console.log(`notInFossaEmails : ${emailsNotRegisteredInFossa}`)
              })
              
              fossaMaintainerIds = buildFossaTeam(projectMaintainersEmailAddrs, teamResponse.id);
              fossa.addMaintainersToFossaTeam(fossaMaintainerIds,fossaTeamId).then(
                (maintainersResponse) => {
                  storeEvent("FOSSA_EVENT", "TEAM_MEMBER_SHIP_UPDATED", teamResponse)
                }
              )
            }
          )
        } else {
          console.log(`Onboarding : missing matainers for ${proposedProjectName} `)
        }
    })
    const jsonIssues = JSON.stringify(issues, null, 2);    
  });
}).on('error', error => {
  console.error(`Failed to retrieve issues: ${error.message}`);
});

/**
 * 
 * @param {*} projectMaintainersEmailAddrs 
 *) @*param 
 */
function buildFossaTeam(projectMaintainersEmailAddrs, fossaTeamId) {
  console.log(`buildFossaTeam(${projectMaintainersEmailAddrs},${fossaTeamId})`);
  
  projectMaintainersEmailAddrs.forEach(maintainerEmail => {
    maintainerId = fossa.getFossaUserByEmail(maintainerEmail);
    if (maintainerId !== null) {
      fossaMaintainerIds.push(maintainerId);
    } else {
      fossa.sendInviteToUser(maintainerEmail);
    }
  });
  return fossaMaintainerIds;
}

function storeEvent(eventType, action, data) {
  const ts = Date.now()
  const e = `${ts}, ${eventType}, ${action}, ${data}`
  const auditFile = 'fossa-log.json'
  writeFile(auditFile, e, 'utf-8').then()
      .catch(error => {
        console.error(`Failed to emit event, ${e} to log: ${error.message}`);
      }); 
}

/**
 * For now, extract new Project Name from Issue Title in cncf/toc issue.
 * @param {string} title 
 * @returns 
 */
function getProjectName(title){
    var result = title.split("[SANDBOX PROJECT ONBOARDING]")
    return result[1].trim()
}
async function addLabelToIssue(issueNumber, label) {
  const options = {
    hostname: 'api.github.com',
    path: `/repos/${owner}/${repo}/issues/${issueNumber}/labels`,
    method: 'POST',
    headers: {
      'User-Agent': 'node.js',
      Authorization: `token ${githubToken}`,
      'Content-Type': 'application/json',
      Accept: 'application/vnd.github.v3+json',
    },
  };

  const labelPayload = JSON.stringify([label]);

  return new Promise((resolve, reject) => {
    const req = https.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => {
        data += chunk;
      });
      res.on('end', () => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(JSON.parse(data));
        } else {
          reject(new Error(`Failed to add label to GitHub issue: ${res.statusCode} - ${data}`));
        }
      });
    });
    req.on('error', (err) => {
      reject(err);
    });
    req.write(labelPayload);
    req.end();
  });
}
