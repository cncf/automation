import { google, sheets_v4 } from 'googleapis';

let teamMaintainerMap = new Map();
const range = 'Active'; // Name of the sheet we want to process 

/**
 * Returns a map of CNCF Projects to an array of maintainers email addresses  
 * @param {*} spreadsheetId 
 * @returns 
 */
export async function getCommunityMaintainerMap(spreadsheetId) {
  if (teamMaintainerMap.keys().length > 0)
    return teamMaintainerMap;
  const keyPath = './service-account-key.json'; // NB Path relative to dir where node invoked
  
  const auth = new google.auth.GoogleAuth({
    keyFile: keyPath,
    scopes: ['https://www.googleapis.com/auth/spreadsheets.readonly'],
  });

  const sheets = google.sheets({ version: 'v4', auth: auth });

  const response = await sheets.spreadsheets.values.get({
    spreadsheetId,
    range,
  });

  const rows = response.data.values;
  
  var  currentProject = '';

  rows.forEach((row) => {  
    const project = row[1]; 
    if (project !== '') {
        currentProject = project
    }
    const maintainer = `${row[2]} <${row[5]}>`;

    if (!teamMaintainerMap.has(currentProject)) {
      teamMaintainerMap.set(currentProject, []);
    }

    teamMaintainerMap.get(currentProject)?.push(maintainer);
  });

  return teamMaintainerMap;
}