import * as dotenv from 'dotenv' // see https://github.com/motdotla/dotenv#how-do-i-use-dotenv-with-import
dotenv.config()

// TODO Mmigrate from JS to TS
// TODO read module data at run time
const nsmodule =  "@cncf-community.fossa"

// TODO Nascent Process Automation Ops Analytics (AWS SNS)
const tick = Date.now()
const cncfTrace = (msg) => console.log(`"Elapsed: ${Date.now() - tick}ms\t","${nsmodule}","${msg}"`);

// TODO Generate a Typescript client for FOSSA??
// https://app.swaggerhub.com/apis-docs/FOSSA1/App/0.3.8
const cncfOrganizationId = 162     
const fossaRoot ='https://app.fossa.com/api'
const fossaTeamsEndPoint =`${fossaRoot}/teams`
const fossaUsersEndPoint =`${fossaRoot}/users`
const fossaOrgInviteEndPoint = `${fossaRoot}/organizations/${cncfOrganizationId}/invite`
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
 * Sends a Fossa User invite out to email
 * 
 * @param {*} emails - array of email addrs 
 * @returns true if sucessfullyt sent error other wise
 */
export async function sendFossaUserInvite(emails) {
    const invocation = `sendFossaUserInvite(${emails})`

    var requestOptions = {...options, method:'POST'}
    var inviteEndpoint =`${fossaOrgInviteEndPoint}`
    
    try {
        cncfTrace (JSON.stringify([invocation, requestOptions.method, inviteEndpoint ]))
        const response = await fetch(`${inviteEndpoint}`, requestOptions);
        fossaUsersCache = await response.json();
        if (response.ok) {
            cncfTrace (JSON.stringify([invocation, requestOptions.method, inviteEndpoint, `Sent invites to ${emails.join()}` ]))
        } else {
            cncfTrace (JSON.stringify([invocation, requestOptions.method, inviteEndpoint, `Status : ${response.status} ${response.statusText}`]))
        }
        return response.ok
    } catch (err) {
        cncfTrace(`${invocation} : ${err}`)
        throw new Error(err);
    }
}

/**
 * Returns a Fossa User account that corresponds to email
 * Otherwise returns undefined if the account does not exist.
 * 
 * @param {*} email 
 * @returns 
 */
export async function getFossaUserByEmail(email) {
    const invocation = `getFossaUserByEmail(${email})`

    var requestOptions = {...options, method:'GET'}
    var userFilteredByEmailAddrQuery =`${fossaUsersEndPoint}`
    
    try {
        if (fossaUsersCache.length == 0) {
            cncfTrace ("UserCache empty")
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

/*
 * @param {string} fossaTeamName 
 * @returns if fossaTeamName is found in FOSSA returns a 
 * Promise that will yield the FOSSA Team MetaData otherwise 
 * null  
 */
export async function getFossaTeamFromCache(fossaTeamName) {
    const invocation = `getFossaTeamFromCache(${fossaTeamName})`
    var fossaTeamId = null
    
    var requestOptions = {...options, method:'GET'}
    cncfTrace (JSON.stringify([invocation, requestOptions, fossaTeamsEndPoint]))

    try {
        if (fossaTeamsCache.length == 0) {
            cncfTrace (JSON.stringify([invocation, requestOptions.method, fossaTeamsEndPoint ]))
            const response = await fetch(`${fossaTeamsEndPoint}`, requestOptions);
            fossaTeamsCache = await response.json();
        }
        const filteredTeam = fossaTeamsCache.filter(t => t.name === fossaTeamName);
        var fossaTeam = filteredTeam;

        if (fossaTeam[0] !== undefined){ //
            cncfTrace(Object.entries(fossaTeam))
            for (const [key, value] of Object.entries(fossaTeam)) {
                cncfTrace(`${key}: ${value}`);
            }
            fossaTeam = filteredTeam[0];
        } 
        return fossaTeam
    } catch (err) {
        console.error(err);
        return null; 
    }
}

export async function isExistingTeamMember(emailAddr,teamId) {
    cncfTrace(`${fossaTeamsCache[teamId][emailAddr]}`)
    return fossaTeamsCache[teamId]
}

/**
 * 
 * @param {*} fossaTeamName 
 * @returns fossaTeamId 
 */
export async function getFossaTeamId(fossaTeamName) {
    // cache look up for the fossaTeamName 
    return await getFossaTeamFromCache(fossaTeamName).id
}
/**
 * 
 * @param {*} fossaTeamName 
 * @returns array of team members
 */
export async function getFossaTeamMembers(fossaTeamName) {
    // cache look up for the fossaTeamName 
    return await getFossaTeamFromCache(fossaTeamName).users
}

export async function createFossaTeam(fossaTeamName) {
    const invocation = `createFossaTeam(${fossaTeamName})`

    var requestOptions = {
        ...options,
        method : "POST",
    }
    var reqBody = {
        name: fossaTeamName,
        autoAddUsers : false,
        defaultRoleId: 4,
        // TODO ?? uniqueIdentifier: canonicalProjectName 
        //      codified CNCF project name 
        //      (deCamelCase => de-camel-cale, space => dash, remove braces) 
    }
    requestOptions.body = JSON.stringify(reqBody)
    var fossaTeamsEndpoint = `${fossaTeamsEndPoint}`
    cncfTrace(JSON.stringify(fossaTeamsEndpoint,requestOptions))

    try {
        var response = await fetch(fossaTeamsEndpoint, requestOptions)
        if (response.ok){
            fossaTeamsCache.push(response.data)
            return response.json()
        } else {
            throw new Error(`${invocation} Error : submitting ${fossaTeamsEndpoint} ${requestOptions.body} => ${response.status} : ${response.statusText}`)
        }
    } catch (err) {
        console.error(err); // fetch only errors out if there is a runtime problem issuing the req
        return null; // Or you can throw an error if you want to handle it differently
    }
}

/**
 * Adds maintainers array FOSSA team granting them Team Admin 
 * - by convention there is a one-to-one mapping between a CNCF Project and a
 * - Team on Fossa
 * For now I will hard code the roleId to be 4 which is a TeamAdmin
 * @param {*} maintainers array of fossaUser/fossaUserId s
 * @param {*} fossaTeamId 
 * @returns 
 * @ref https://app.swaggerhub.com/apis-docs/FOSSA1/App/0.3.8#/Users/putTeamUsers
*/
export async function addMaintainersToFossaTeam(maintainers,fossaTeamId) {
    const invocation = `addMaintainersToFossaTeam(${maintainers}, ${fossaTeamId})`
    
    var newMaintainers = []
        
    for (const [, user] of Object.entries(maintainers)) {
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

export function addCollaboratorToFossaTeam() {
    return "TODO : Implement addCollaboratorToFossaTeam"
}

export async function getMaintainers(userEmails) {
    maintainerFossaUsers = await Promise.all(userEmails.map(email => getFossaUserByEmail(email)));
    return maintainerFossaUsers;
}

(fossaUsersCache.length>0) ? cncfTrace(`${fossaUsersCache.length} Users Cached!`)
   : cncfTrace(`FOSSA User Cache is Empty!`)
// getMaintainers(maintainerEmails)
//     .then(
//         maintainers =>  getFossaTeam("CNCF Process Automation").then(
//             (cncfProccessTeamId) => {
//                 cncfTrace(`teamId : ${cncfProccessTeamId}`);
//                 const team = fossaTeamsCache.filter(team => team.id === cncfProccessTeamId);
//                 cncfTrace(`team : ${team[0].name}`);
//                 addMaintainersToFossaTeam(maintainers, cncfProccessTeamId);
//             }
//         )
//     )
//     .catch(error => console.error(error));