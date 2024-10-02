import { writeFile } from 'fs';

// TODO read at run time?
const nsmodule =  "@cncf-community.jira"

// TODO Process Automation Analytics - ref AWS SNS 
const tick = Date.now()
const cncfTrace = (msg) => console.log(`"Elapsed: ${Date.now() - tick}ms\t","${nsmodule}","${msg}"`);
const cncfEvent = (msg) => console.log(`"Elapsed: ${Date.now() - tick}ms\t","${nsmodule}","${msg}"`);

const jiraRoot ='https://app.jira.com/api' // Note: app api placement and usage!
const jiraTeamsEndPoint =`${jiraRoot}/teams`
const jiraUsersEndPoint =`${jiraRoot}/users`

var jiraTeams = []
var jiraUsers = []
/**
 * 
 * @returns 
 */
function getjiraToken() {
    if (typeof process.env.JIRA_TOKEN === 'undefined'){
        console.error(
`jira.js/ndex.js : ERROR JIRA_TOKEN environment variable not set.
jira.js/ndex.js : https://developer.atlassian.com/cloud/jira/platform/basic-auth-for-rest-apis/
jira.js/ndex.js : Exiting.`
        )
        process.exit(1)
    } else {
        cncfTrace(`JIRA_TOKEN set to ${process.env.jira_TOKEN}`)
        return process.env.jira_TOKEN
    }
}

// Set options for Fetch API requests
const options = {
    method: 'GET',
    headers: {
        'Authorization': `Bearer 7772e5b485998c766f7f8d975065400f`,
        'Accept': 'application/json',
        'Content-Type': 'application/json',
    }
};
async function getjiraUsers() {
    const invocation = `getjiraUsers()`

    var requestOptions = {...options, method:'GET'}
    cncfTrace (JSON.stringify([invocation, requestOptions, jiraUsersEndPoint]))

    await fetch(jiraUsersEndPoint, requestOptions)
    .then(response => {
        return response.json().then(Users => {
            Users.forEach((user) => {
                // cncfTrace(user.username)
                jiraUsers.push(user) 
            })
        }), err => {console.error(err)}})
        .finally(cncfTrace("Loaded users into jiraUsers" + jiraUsers.length))
    .catch(err => console.error(err))
    
}

async function getjiraUserByEmail(email) {

    // Make Fetch API request to jira API
    var requestOptions = {...options, method:'GET'}
    cncfTrace (JSON.stringify([invocation, requestOptions.method, jiraUsersEndPoint]))
    
    await fetch(`${jiraUsersEndPoint}? encodeURIComponent(${email})`, options)
    .then(response => { (response.ok) ? return response.json(): return , err => {console.error(err)}})
    .catch(err => console.error(err))
}

async function getjiraTeams() {
    const invocation = `getjiraTeams()`

    // Make Fetch API request to jira API
    var requestOptions = {...options, method:'GET'}
    cncfTrace (JSON.stringify([invocation, requestOptions.method, jiraTeamsEndPoint]))

    await fetch(jiraTeamsEndPoint, requestOptions)
    .then(response => {
        return response.json().then(teams => {
            teams.forEach((team) => {
                // cncfTrace(team.name)
                jiraTeams.push(team)
            })
        }), err => {console.error(err)}})
    .catch(err => console.error(err))
}
/**
 * Creates jira Team nammed cncfProject 
 * @param {*} cncfProject 
 */
async function createjiraTeam(cncfProject){
    const invocation = `createjiraTeam(${cncfProject})`
    cncfTrace(`${invocation}`)
    
    var requestOptions = {
        ...options, 
        method:'POST',
        name : cncfProject,
        autoAddUsers: true, // TODO log a call with jira Support about this. Docs request.
        teamUsers: null,
        teamProjects: null,
    }
    requestOptions.body = JSON.stringify(reqBody)

    var promise = fetch(jiraTeamsEndPoint, requestOptions)
    .then(res => res.json())
    .then(data => cncfTrace("created:" + data))
    .catch(function(error){
        const jiraError = new Error(`${invocation} failed.` 
                            + `\n\trequestOptions at runtime: ${requestOptions}`
                            + `\n\tAxios Error: ${error}`)
        cncfTrace(`${invocation} ERROR during Create Team ${jiraError}`)
    }).finally(function(){
        cncfTrace(`${invocation} finally called.`)
    })
    return promise
}
/*
 * Adds named maintainers to jira team granting them Team Admin 
 * - there is a one-to-one mapping between a CNCF Project a Team on jira
 *   though that is not enforced in this function
 * @param {[{id,roleId}}]} maintainer Array of fossUser objects  {
      "id": 0,
      "roleId": 0
    }
 * @param {team} team (corresponds to a CNCF Project)
 * @ref https://app.swaggerhub.com/apis-docs/jira1/App/0.3.8#/Users/putTeamUsers
 */
function addMaintainersTojiraTeam(maintainers,cncfProject) {
    const invocation = `addMaintainersTojiraTeam(${maintainers}, ${cncfProject.id})`
    cncfTrace(`${invocation}`)
    
    var requestOptions = {
        ...options,
        method : "PUT",
        body : {
            action : "add",
            users : maintainers
        }
    }
    JSON.stringify(requestOptions.body)
    cncfTrace(JSON.stringify(jiraTeamsEndPoint,options))

    const promise = fetch(jiraTeamsEndPoint + `/${cncfProject.id}/users`, options) 
    .then(response => {
        if (!response.ok){
            throw new Error(`${invocation} response not ok`)
        }
        return res.json()
    }
    ).catch(function(error){
        const jiraError = new Error(`${invocation} fetch(${jiraTeamsEndPoint},${options}) failed.`
                            + `\n\tFetch Error: ${error}`)
        cncfTrace(`${invocation} ${jiraError}`)
    }).finally(function(){
        cncfTrace(`${invocation} finally called.`)
    })
    return promise
}

function addCollaboratorTojiraTeam(maintainer,team) {
    return "TODO"
}

const jiraToken =  getjiraToken()
var maintainer = await getjiraUserByEmail("robert.kielty@cncf.io")
cncfTrace("MAINTAINER" + JSON.stringify(maintainer,null,2))
process.exit(1)
getjiraTeams()
.then(getjiraUsers())
.then(createjiraTeam("CNCF Process Automation"))
.then(
    () => {
        var cncfProcAutoTeam = jiraTeams.find(team => team.name==="CNCF Process Automation")
        var maintainer = getjiraUserByEmail("robert.kielty@cncf.io")
        cncfTrace("MAINTAINER" + JSON.stringify(maintainer,null,2))
        // cncfTrace("TEAM\n" + JSON.stringify(cncfProcAutoTeam,null,2) + "\nTEAM")
        // cncfTrace("USER\n" + JSON.stringify(jiraUsers[2],null,2) + "\nUSER")
        // addMaintainersTojiraTeam(["robert.kielty@cncf.io","jsica@linuxfoundation.org","ihor@linux.com"], cncfProcAutoTeam)
})
