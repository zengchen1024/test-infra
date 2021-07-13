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

### Directory Owners (Part One)
Directory Owners defines the approvers and reviewers for any files in a directory except the ones defined in Part Two of OWNERS file

It will find approvers and reviewers from OWNERS file of current and parent directories by default, excpet the `no_parent_owners` is set as true.

If `no_parent_owners` is true, then it will not retrieve the approvers or reviewers from parent directories when the Part One defines them correspondingly.

However, for the following OWNERS file, it will only find approvers from current directory, but for reviewers it will not, because reviewers is undefined.

```yaml
approvers:
- bob
options:
  no_parent_owners: true
```

### File Owners (Part Two)
It can defines special approvers or reviewers for some files in a directory in Part Two of OWNERS file.

If the file name matches to the key defined in `files`, then it will find approvers and reviewers from it.

If the file name matches the key but approvers or reviewers are not defined, it will find them from Part One of OWNERS file in current and parent directories.

**Note**:

There is not `no_parent_owners` field in Part Two, because this part is for special files and parent OWNERS files are ignored by default.
