package main

import "os/user"

const userColor = "[green:]"
const rootColor = "[red:]"

func UsernameWithColor(username string) string {
	user, err := user.Lookup(username)
	if err != nil {
		return userColor + username
	}

	if user.Uid == "0" {
		return rootColor + username
	}

	return userColor + username
}

func GroupnameWithColor(groupname string) string {
	group, err := user.LookupGroup(groupname)
	if err != nil {
		return userColor + groupname
	}

	if group.Gid == "0" {
		return rootColor + groupname
	}

	return userColor + groupname
}
