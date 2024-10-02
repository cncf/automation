import { writeFile } from 'fs';
// TODO Mmigrate from JS to TS
// TODO read module data at run time
const nsmodule =  "@cncf-community.fossa"

// TODO Nascent Process Automation Ops Analytics (AWS SNS)
const tick = Date.now()
const cncfTrace = (msg) => console.log(`"Elapsed: ${Date.now() - tick}ms\t","${nsmodule}","${msg}"`);
const cncfEvent = (result, operation, data, service) => console.log(`"Elapsed: ${Date.now() - tick}ms\t","${nsmodule}", "${result}, ${operation}, ${data}, ${service} "`);

// TODO Generate a Typescript client for FOSSA 
const fossaRoot ='https://app.fossa.com/api'
const fossaTeamsEndPoint =`${fossaRoot}/teams`
const fossaRolesEndPoint =`${fossaRoot}/roles`
const fossaUsersEndPoint =`${fossaRoot}/users`

var fossaTeamsCache = []
var fossaUsersCache = []

function getFossaToken() {
    if (typeof process.env.FOSSA_TOKEN === 'undefined'){
        console.error(`ERROR FOSSA_TOKEN env var not set. ref: https://docs.fossa.com/docs/api-reference#api-tokens fossa.js/ndex.js : Exiting.`)
        process.exit(1)
    } else {
        return process.env.FOSSA_TOKEN
    }
}

// Set options for Fetch API requests
const options = {
    method: 'GET',
    headers: {
        'Authorization': `Bearer ${getFossaToken()}`,
        'Accept': 'application/json',
        'Content-Type': 'application/json',
    }
};

/**
 * Returns a Fossa User account that corresponds to email
 * Otherwise returns undefined if the account does not exist.
 * 
 * @param {*} email 
 * @returns 
 */
async function getFossaUserByEmail(email) {
    const invocation = `getFossaUserByEmail(${email})`

    var requestOptions = {...options, method:'GET'}
    var userFilteredByEmailAddrQuery =`${fossaUsersEndPoint}`
    
    try {
        if (fossaUsersCache.length == 0) {
            cncfTrace ("EMPTY CACHE!!!!")
            cncfTrace (JSON.stringify([invocation, requestOptions.method, userFilteredByEmailAddrQuery ]))
            const response = await fetch(`${userFilteredByEmailAddrQuery}`, requestOptions);
            fossaUsersCache = await response.json();
        }
        cncfTrace (`CACHE LOOKUP for ${email}`)
        var filteredUser = fossaUsersCache.filter(u => u.email === email);
        cncfTrace (`Found ${filteredUser[0].username}`)
        return Promise.resolve(filteredUser[0]);
    } catch (err) {
        cncfTrace(`${invocation} : ${err}`)
        throw new Error(err);
    }
}

async function getFossaTeam(fossaTeamName) {
    const invocation = `getFossaTeams(${fossaTeamName})`

    var requestOptions = {...options, method:'GET'}
    cncfTrace (JSON.stringify([invocation, requestOptions, fossaTeamsEndPoint]))

    try {
        if (fossaTeamsCache.length == 0) {
            cncfTrace (JSON.stringify([invocation, requestOptions.method, fossaTeamsEndPoint ]))
            const response = await fetch(`${fossaTeamsEndPoint}`, requestOptions);
            fossaTeamsCache = await response.json();
        }
        const filteredTeam = fossaTeamsCache.filter(t => t.name === fossaTeamName);
        
        return Promise.resolve(filteredTeam[0].id);
    } catch (fossaTeamsEndPoint) {
        console.error(err);
        return null; 
    }
}

/**
 * Adds maintainers array to fossTeamIfd FOSSA team granting them Team Admin 
 * - by convention there is a one-to-one mapping between a CNCF Project and a
 * - Team on Fossa
 * For now I will hard code the roleId to be 4 which is a TeamAdmin
 * @param {*} maintainers array of fossaUser/fossaUserId s
 * @param {*} fossaTeamId 
 * @returns 
 * @ref https://app.swaggerhub.com/apis-docs/FOSSA1/App/0.3.8#/Users/putTeamUsers
*/
async function addMaintainersToFossaTeam(maintainers,fossaTeamId) {
    const invocation = `addMaintainersToFossaTeam(${maintainers}, ${fossaTeamId})`
    
    var newMaintainers = []
        
    for (const [key, user] of Object.entries(maintainers)) {
        newMaintainers.push(new Object({id: user.id, roleId: 4}) )
    }

    var requestOptions = {
        ...options,
        method : "PUT",
    }

    var reqBody = {
        action : "add",
        users : newMaintainers
    }

    requestOptions.body = JSON.stringify(reqBody)
    var fossaTeamEndpoint = `${fossaTeamsEndPoint}/${fossaTeamId}/users`
    cncfTrace(JSON.stringify(fossaTeamEndpoint,requestOptions))

    try {
        var response = await fetch(fossaTeamEndpoint, requestOptions)
        if (response.ok){
            return response.json()
        } else {
            throw new Error(`${invocation} Error : submitting ${fossaTeamEndpoint} ${requestOptions.body} => ${response.status} : ${response.statusText}`)
        }
    } catch (err) {
        console.error(err); // fetch only errors out if there is a runtime problem issuing the req
        return null; // Or you can throw an error if you want to handle it differently
    }
}

function addCollaboratorToFossaTeam(maintainer,team) {
    return "TODO "
}

const fossaToken =  getFossaToken()

var maintainerEmails = ["robert.kielty@cncf.io", "jsica@linuxfoundation.org", "ihor@linux.com"];
var maintainerFossaUsers = [];
// Use Promise.all to wait for all promises to resolve


async function getMaintainers(userEmails) {
    maintainerFossaUsers = await Promise.all(userEmails.map(email => getFossaUserByEmail(email)));
    return maintainerFossaUsers;
}

// (fossaUsersCache.length>0) ? cncfTrace(`${fossaUsersCache.length} Users Cached!`)
//   : cncfTrace(`FOSSA User Cache is Empty!`)
getMaintainers(maintainerEmails)
    .then(
        maintainers =>  getFossaTeam("CNCF Process Automation").then(
            (cncfProccessTeamId) => {
                cncfTrace(`teamId : ${cncfProccessTeamId}`);
                const team = fossaTeamsCache.filter(team => team.id === cncfProccessTeamId);
                cncfTrace(`team : ${team[0].name}`);
                addMaintainersToFossaTeam(maintainers, cncfProccessTeamId);
            }
        )
    )
    .catch(error => console.error(error));