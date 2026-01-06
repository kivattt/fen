Remember the `-y` flag to allow zipping symlinks
```
zip -y -r files.zip * .git .gitignore
```

## Structure of .zip tests

Our only requirement for something to be a git repository is for there to be a .git folder.
This is inconsistent with `git status`, which presumably checks for some other files to consider it a repository.

Because of this, we need tests to atleast have an empty .git folder in them (unless they're specifically checking if a .git folder is missing AKA not-a-repository).
An empty .git folder is like a brand new `git init` repository, because it is missing the `.git/index` file, meaning all files are untracked.

Most tests should have a `.git/index` file.

## How to create an "empty" .zip file

The `tests-statusraw/2_missing_files/files.zip` file is an empty zip file generated with this:\
https://stackoverflow.com/a/64466237

Contents of that link, copied below:
```
I have done it the dirty way:

Created an file called emptyfile with one byte in it (whitespace).

Then I added that file into the zip file:

zip -q {{project}}.zip emptyfile

To finally remove it:

zip -dq {{project}}.zip emptyfile

(if -q is omitted, you will get warning: zip warning: zip file empty)

This way I've got an empty zip file one can -update.

All that can be converted to a oneliner (idea from GaspardP):

echo | zip -q > {{project}}.zip && zip -dq {{project}}.zip -

Is there a more elegant way to do this?
```
