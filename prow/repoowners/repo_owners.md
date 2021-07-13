## The struct of OWNERS file

A OWNERS file includes two parts generally and both of them are optional.

```yaml
# (Part One) This part is active for all files in a directory 
approvers:
- bob
reviewers:
- alice
options:
  no_parent_owners: true

# (Part Two) This part is active for some special files in a directory
files:
  "\\.go$": # key is in format of regular expression
    approvers:
    - tony
    reviewers:
    - elvis
```

### Directory Owners
Directory Owners is Part One defined in a OWNERS file which means the approvers and reviewers will be found from it for any files except the ones defined in Part Two in the directory. 

Note:

If `no_parent_owners` is true, then it will not retrieve the approvers or reviewers from parent directories when the Part One defined approvers and reviewers correspondingly.

However, for the following OWNERS file, it will only find approvers from current directory, but for reviewers it will not.

Because there is not reviewers in OWNERS file, and the `no_parent_owenrs` is unactive to retrieve reviewers.

```yaml
approvers:
- bob
options:
  no_parent_owners: true
```

### File Owners
It can defines speeical approvers or reviewers for some files in a directory.

If the file name matches to the key defined in `files` in Part Two, then it will find approvers and reviewers from it.

Note: 

There is not `no_parent_owners` field in Part Two, because this part is for special files.

If the file name matches the key but approvers or reviewers are not defined, it will also find them from current and parent directories.
