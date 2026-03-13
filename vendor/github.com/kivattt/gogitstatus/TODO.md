- Implement Git submodules .gitmodules / .git regular file pointing with "gitdir: ...". Folders on my laptop with this issue: `projects/PYNQ` `projects/clap-tutorials/libs/clap` `projects/learning_odin/14_shared_object/cmake-sfml-project`
`14_shared_object/cmake-sfml-project` is weird, it has no .git file but still theres a problem.

- We already fixed skipping "/" in goignore, but "/////////" should also be skipped, make sure that happens.
- use mywalkdir / myreaddir in fen aswell to remove the unnecessary sorting overhead
- in goignore: make fast path when pattern has no wildcards?
- Make test output better (horizontal 1, 2, 3... instead of taking up so many lines, or omitting successful tests)


- (BAD) doing twice the work by checking if parent folders are ignored
- (BETTER) only do that work on folders AND the first entry per thread
- (BEST) fix it in goignore which would be the right thing to do.
         it won't be that slow because we use skipDir()

// keep track of ignored folders in a list/map.

if negation pattern:
	if any parent folder ignored in known list/map lookup
		skip line



## TODO
- Deal with .git files that point the real .git folder elsewhere (submodules or something)
- Support exclude file priority (like core.excludesFile in config and other XDG\_CONFIG stuff)
- Support SHA-256
- Support other Git Index versions besides 2
- Deal with .gitattributes (and XDG\_CONFIG stuff) to determine whether we need to hash with line endings normalized. See: `tests-status/36_line_ending_conversion_during_hash/README.md`
