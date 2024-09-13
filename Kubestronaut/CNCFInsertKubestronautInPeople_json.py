import csv
import json
import gdown
import os
from collections import OrderedDict
import argparse
import shutil

# In the same directory a file named Kubestronaut.tsv should contains the export
# of the Kubestronauts responses in tsv
# That script should access the people CNCF repo with the following path ../../people
parser = argparse.ArgumentParser(description='Add Kubestronaut to the people.json file')
parser.add_argument('-fl','--firstLine', help='First row number to be added from the tsv file', required=True)
parser.add_argument('-ll','--lastLine', help='Last row number to be added from the tsv file', required=True)
args = vars(parser.parse_args())

firstLineToBeInserted = int(args['firstLine'])
lastLineToBeInserted = int(args['lastLine'])

class People:
    def __init__(self, name, bio, company, pronouns, location, linkedin, twitter, github, wechat, website, youtube, slack_id, image):
        self.name=name
        self.bio="<p>"+bio.replace("   ","<p/><p>")+"</p>"
        self.company=company
        self.pronouns=pronouns
        self.location=location

        if linkedin.startswith(("https","http")):
            self.linkedin=linkedin
        elif linkedin:
            self.linkedin="https://www.linkedin.com/in/"+linkedin
        else:
            self.linkedin=""

        if twitter.startswith(("https","http")):
            self.twitter=twitter
        elif twitter :
            self.twitter="https://twitter.com/"+twitter
        else:
            self.twitter=""

        if github.startswith(("https","http")):
            self.github=github
        elif github:
            self.github="https://github.com/"+github
        else:
            self.github=""

        if wechat.startswith(("https","http")):
            self.wechat=wechat
        elif wechat:
            self.wechat="https://web.wechat.com/"+wechat
        else:
            self.wechat=""

        self.website=website

        if youtube.startswith(("https","http")):
            self.youtube=youtube
        elif youtube:
            self.youtube="https://www.youtube.com/c/"+youtube
        else:
            self.youtube=""

        self.category=["Kubestronaut"]
        self.slack_id=slack_id

        if (image) :
            url = image
            gdown.download(url, "imageTemp.jpg", fuzzy=True, quiet=False)
            output=name.lower().replace(" ","-")+".jpg"
        else :
            shutil.copy("phippy.jpg","imageTemp.jpg")
            output="phippy.jpg"
        self.image=output

    def toJSON(self):
        return json.dumps(
            self,
            default=lambda o: o.__dict__, 
            indent=4)

# Retrieve JSON data from the file
with open('../../people/people.json', "r+") as jsonfile:
#    print(jsonfile.read())
    data = json.load(jsonfile)


for lineToBeInserted in range(firstLineToBeInserted, lastLineToBeInserted+1, 1):

    # Import CSV that needs to be treated
    with open('Kubestronaut.tsv') as csv_file:
        lineCount = 1
        csv_reader = csv.reader(csv_file, delimiter='\t')
        peopleFound=False
        for row in csv_reader:
            if lineCount == lineToBeInserted:
                print(f'\t{row[1]}')
                newPeople = People(name=row[1], bio=row[2], company=row[3], pronouns=row[4], location=row[5], linkedin=row[6], twitter=row[7], github=row[8], wechat=row[9], website=row[10], youtube=row[11], slack_id=row[13], image=row[14])
                peopleFound=True
                break
            else:
                lineCount += 1
        if (peopleFound == False):
            print("People not Found "+row[1]+", abort !")
            exit(1)


    print(newPeople.toJSON())

    indexPeople=0
    for people in data:
        #print(people["name"])
        if people["name"].lower() < newPeople.name.lower():
            indexPeople += 1
            continue
        if people["name"].lower() == newPeople.name.lower():
            print("{newPeople.name} already in people.json, abort !")
            exit(2)
        else:
            print(people['name']+' et '+newPeople.name)
            data.insert(indexPeople, json.JSONDecoder(object_pairs_hook=OrderedDict).decode(newPeople.toJSON()))
            os.rename("imageTemp.jpg", "people/images/"+newPeople.image)
            break


with open('../../people/people.json', "r+", encoding='utf-8') as jsonfile:
    jsonfile.write(json.dumps(data, indent=3, ensure_ascii=False, sort_keys=False))
