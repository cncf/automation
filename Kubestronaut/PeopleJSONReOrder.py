import json

#Variables

# Decide the two file paths according to your 
# computer system
jsonPeople = '../../people/people.json'
jsonPeople2 = 'people3.json'
# Open a json writer, and use the json.dumps() 
# function to dump data
with open(jsonPeople, 'r', encoding='utf-8') as peopleFile:
    people  = json.load(peopleFile)
    for ppl in people:
        ppl['name']=ppl['name'].strip().title()
    sorted_people = sorted(people, key=lambda x: x['name'])

with open(jsonPeople2, 'w', encoding='utf-8') as jsonf:
    jsonf.write(json.dumps(sorted_people, indent=4, ensure_ascii=False))
