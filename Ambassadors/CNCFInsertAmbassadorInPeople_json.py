import csv
import json
import os
import re
from collections import OrderedDict
import argparse
import shutil
import io
from difflib import get_close_matches
from google.oauth2.credentials import Credentials
from google_auth_oauthlib.flow import InstalledAppFlow
from googleapiclient.discovery import build
from googleapiclient.http import MediaIoBaseDownload
from google.auth.transport.requests import Request

# Constants
GDRIVE_URL = "https://drive.google.com/drive/folders/XXXXXXXX"
PROJECT_LIST_PATH = 'CNCF-Project-list.txt'
PEOPLE_JSON_PATH = '../../people/people.json'
SUGGESTIONS_FILE = 'suggestions.json'

# Load existing suggestions or create a new dictionary if the file doesn't exist
def load_suggestions():
    if os.path.exists(SUGGESTIONS_FILE):
        with open(SUGGESTIONS_FILE, 'r', encoding='utf-8') as file:
            return json.load(file)
    return {}

def save_suggestions(suggestions):
    with open(SUGGESTIONS_FILE, 'w', encoding='utf-8') as file:
        json.dump(suggestions, file, indent=4)

# Load the existing suggestions
suggestions = load_suggestions()

def load_project_names(filepath):
    """Load project names from a given file into a list."""
    with open(filepath, 'r', encoding='utf-8') as file:
        return [line.strip() for line in file if line.strip()]

def clean_word(word):
    """Remove common punctuation from the word."""
    return word.strip().strip('.,')

def parse_projects(raw_string, project_names):
    """Parse project names from a string using a list of known project names."""
    project_dict = {project.lower(): project for project in project_names}
    words = raw_string.split()
    found_projects = []
    i = 0

    while i < len(words):
        if words[i] == '-':  # Skip hyphens
            i += 1
            continue
        
        match_found = False
        for j in range(len(words), i, -1):
            candidate = ' '.join(clean_word(word) for word in words[i:j]).lower()
            if candidate in project_dict:
                if project_dict[candidate] not in found_projects:
                    found_projects.append(project_dict[candidate])
                i = j  # Skip to the end of the current matched phrase
                match_found = True
                break
        if not match_found:
            i += 1  # Only increment if no match was found

    # Handle remaining unmatched words
    remaining_words = ' '.join(words).lower()
    for project in found_projects:
        remaining_words = remaining_words.replace(project.lower(), '')

    remaining_words = remaining_words.split()

    for i in range(len(remaining_words)):
        for j in range(i+1, len(remaining_words)+1):
            word_sequence = ' '.join(clean_word(word) for word in remaining_words[i:j])
            if word_sequence in suggestions:
                if suggestions[word_sequence] and suggestions[word_sequence] not in found_projects:
                    found_projects.append(suggestions[word_sequence])
                break
            close_matches = get_close_matches(word_sequence, project_dict.keys(), n=1, cutoff=0.8)
            if close_matches:
                suggested = project_dict[close_matches[0]]
                response = input(f"Did you mean '{suggested}' for '{word_sequence}'? (Y/n) or type the correct name: ")
                if response.lower() == 'y':
                    if suggested not in found_projects:
                        found_projects.append(suggested)
                    for k in range(i, j):
                        remaining_words[k] = ''
                    suggestions[word_sequence] = suggested
                    save_suggestions(suggestions)
                    break
                elif response.strip():
                    if response.strip() not in found_projects:
                        found_projects.append(response.strip())
                    for k in range(i, j):
                        remaining_words[k] = ''
                    suggestions[word_sequence] = response.strip()
                    save_suggestions(suggestions)
                    break
                else:
                    suggestions[word_sequence] = None
                    save_suggestions(suggestions)
        
        remaining_words = [word for word in remaining_words if word]

    for word in remaining_words:
        if word == '-':  # Skip hyphens
            continue
        if word in suggestions:
            continue
        print(f"No suggestions for '{word}'. Please enter the correct name or press enter to skip:")
        response = input()
        if response.strip():
            if response.strip() not in found_projects:
                found_projects.append(response.strip())
            suggestions[word] = response.strip()
        else:
            suggestions[word] = None
        save_suggestions(suggestions)

    return found_projects

def download_file_from_drive(folder_url, filename, outputname):
    """Download a file from Google Drive."""
    SCOPES = ['https://www.googleapis.com/auth/drive.readonly']
    TOKEN_FILE = 'token.json'
    CREDENTIALS_FILE = 'credentials.json'
    creds = Credentials.from_authorized_user_file(TOKEN_FILE, SCOPES) if os.path.exists(TOKEN_FILE) else None
    if not creds or not creds.valid:
        if creds and creds.expired and creds.refresh_token:
            creds.refresh(Request())
        else:
            flow = InstalledAppFlow.from_client_secrets_file(CREDENTIALS_FILE, SCOPES)
            creds = flow.run_local_server(port=0)
            with open(TOKEN_FILE, 'w') as token:
                token.write(creds.to_json())
    service = build('drive', 'v3', credentials=creds)
    folder_id = folder_url.split('/')[-1]
    results = service.files().list(q=f"'{folder_id}' in parents", spaces='drive', fields="nextPageToken, files(id, name)").execute()
    items = results.get('files', [])
    if not items:
        print('No files found.')
    else:
        for item in items:
            if item['name'] == filename:
                request = service.files().get_media(fileId=item['id'])
                fh = io.BytesIO()
                downloader = MediaIoBaseDownload(fh, request)
                done = False
                while not done:
                    _, done = downloader.next_chunk()
                fh.seek(0)
                with open(outputname, 'wb') as f:
                    f.write(fh.read())
                print(f'Download complete for {filename}')

class People:
    def __init__(self, firstName, lastName, bio, company, pronouns, location, twitter, github, projects, image):
        self.name = f"{firstName} {lastName}"
        self.bio = f"<p>{bio.replace('   ', '<p/><p>')}</p>" if bio else ""
        self.company = company if company != "Individual - No Account" else ""
        self.pronouns = pronouns if pronouns != "Prefer not to answer" else ""
        self.location = location
        self.twitter = self.format_twitter(twitter)
        self.github = self.format_github(github)
        self.category = ["Ambassadors"]
        self.projects = self.parse_and_confirm_projects(projects)
        self.image = self.handle_image(image, firstName, lastName)

    def format_twitter(self, twitter):
        if twitter.startswith("https"):
            return twitter
        elif twitter not in ["n/a", "N/A"]:
            return f"https://twitter.com/{twitter[1:]}" if twitter.startswith("@") else f"https://twitter.com/{twitter}"
        return ""

    def format_github(self, github):
        if github.startswith("https"):
            return github
        elif github not in ["n/a", "N/A"]:
            return f"https://github.com/{github[1:]}" if github.startswith("@") else f"https://github.com/{github}"
        return ""

    def handle_image(self, image, firstName, lastName):
        if image:
            file_extension = os.path.splitext(image)[1]
            download_file_from_drive(GDRIVE_URL, image, "imageTemp" + file_extension)
            return f"{firstName.lower().replace(' ', '-')}-{lastName.lower().replace(' ', '-')}{file_extension}"
        else:
            shutil.copy("phippy.jpg", "imageTemp.jpg")
            return "phippy.jpg"

    def parse_and_confirm_projects(self, raw_string):
        project_names = load_project_names(PROJECT_LIST_PATH)
        project_dict = {project.lower(): project for project in project_names}
        identified_projects = parse_projects(raw_string, project_names)
        
        # Confirm and add any missing projects interactively
        remaining_words = raw_string.lower()
        for project in identified_projects:
            remaining_words = remaining_words.replace(project.lower(), '')

        remaining_words = remaining_words.split()

        for i in range(len(remaining_words)):
            for j in range(i+1, len(remaining_words)+1):
                word_sequence = ' '.join(clean_word(word) for word in remaining_words[i:j])
                if word_sequence in suggestions:
                    if suggestions[word_sequence] and suggestions[word_sequence] not in identified_projects:
                        identified_projects.append(suggestions[word_sequence])
                    break
                close_matches = get_close_matches(word_sequence, project_dict.keys(), n=1, cutoff=0.8)
                if close_matches:
                    suggested = project_dict[close_matches[0]]
                    response = input(f"Did you mean '{suggested}' for '{word_sequence}'? (Y/n) or type the correct name: ")
                    if response.lower() == 'y':
                        if suggested not in identified_projects:
                            identified_projects.append(suggested)
                        for k in range(i, j):
                            remaining_words[k] = ''
                        suggestions[word_sequence] = suggested
                        save_suggestions(suggestions)
                        break
                    elif response.strip():
                        if response.strip() not in identified_projects:
                            identified_projects.append(response.strip())
                        for k in range(i, j):
                            remaining_words[k] = ''
                        suggestions[word_sequence] = response.strip()
                        save_suggestions(suggestions)
                        break
                    else:
                        suggestions[word_sequence] = None
                        save_suggestions(suggestions)
        
        remaining_words = [word for word in remaining_words if word]

        for word in remaining_words:
            if word == '-':  # Skip hyphens
                continue
            if word in suggestions:
                continue
            print(f"No suggestions for '{word}'. Please enter the correct name or press enter to skip:")
            response = input()
            if response.strip():
                if response.strip() not in identified_projects:
                    identified_projects.append(response.strip())
                suggestions[word] = response.strip()
            else:
                suggestions[word] = None
            save_suggestions(suggestions)

        return identified_projects

    def toJSON(self):
        return json.dumps(self, default=lambda o: o.__dict__, indent=4)

def process_entries(firstLine, lastLine):
    with open('Ambassadors.tsv') as csv_file:
        csv_reader = csv.reader(csv_file, delimiter='\t')
        for lineCount, row in enumerate(csv_reader, start=1):
            if firstLine <= lineCount <= lastLine:
                newPerson = People(firstName=row[0], lastName=row[1], bio=row[7], company=row[3], pronouns=row[4],
                                   location=row[2], twitter=row[6], github=row[5], projects=row[8], image=row[9])
                print(newPerson.toJSON())

                # Load existing data
                with open(PEOPLE_JSON_PATH, "r", encoding='utf-8') as jsonfile:
                    data = json.load(jsonfile)

                # Insert new person in sorted order
                indexPeople = 0
                for people in data:
                    if people["name"].lower() < newPerson.name.lower():
                        indexPeople += 1
                        continue
                    if people["name"].lower() == newPerson.name.lower():
                        print(f"{newPerson.name} already in people.json, abort!")
                        exit(2)
                    else:
                        data.insert(indexPeople, json.JSONDecoder(object_pairs_hook=OrderedDict).decode(newPerson.toJSON()))
                        split_tup = os.path.splitext(newPerson.image)
                        os.rename("imageTemp" + split_tup[1], "people/images/" + newPerson.image)
                        break

                # Write updated data back to file
                with open(PEOPLE_JSON_PATH, "w", encoding='utf-8') as jsonfile:
                    json.dump(data, jsonfile, indent=3, ensure_ascii=False, sort_keys=False)

if __name__ == "__main__":
    # Argument Parser Setup
    parser = argparse.ArgumentParser(description='Add Ambassadors to the people.json file')
    parser.add_argument('-fl', '--firstLine', type=int, help='First row number to be added from the tsv file', required=True)
    parser.add_argument('-ll', '--lastLine', type=int, help='Last row number to be added from the tsv file', required=True)
    args = parser.parse_args()

    process_entries(args.firstLine, args.lastLine)
    # Save the updated suggestions
    save_suggestions(suggestions)
