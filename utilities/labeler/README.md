## Label Config / General Notes

When a label configuration is supplied, they should be the only labels that exist unless we set auto-delete to FALSE

When the label configuration is updated (name, color, description etc), it should update the label and all items the label was previously tagged with.

When a label that isn't defined is attempted to be applied, it should not create the label and prompt the user UNLESS we set auto-create to TRUE

When there are multiple matching commands, it should it process all of them.

When the labeler executes, it should only attempt to modify the label state IF the end state is different from the current state e.g. it should not remove and re-add a label if the end condition is the same.

A label should be able to be removed by some method e.g. /remove-<foo> <bar> would remove the label foo/bar or /foo -bar.
No preference, just a method of removing a label needs to exist
  
##  kind/match
  
When the matchList rule is used, it should ONLY execute the actions if the text supplied by the user matches one of the items in the list

When the unique rule is used, only ONE of the defined labels should be present

- This can be renamed / adjusted - essentially need to restrict a set of labels to a 'namespace' and only one can be present in the final state. Maybe this should be processed as soemthing different? definine an end state condition vs matching whats there initially?


## kind/label

When the kind/label rule is used, it should ignore issue/PR bodies and check label state only for taking action.

## kind/filePath
 
Only applies to PRs
 
When a commit changes anything that matches the filepath, the rules defined should execute
  
  
## rules / actions
  
When the remove-label action is present, it should remove the matching label if present

When the apply-label action is used, it should ONLY apply a label if the label exists.