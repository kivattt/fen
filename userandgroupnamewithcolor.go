package main

import "os/user"

const userColor = "[green:]"
const rootColor = "[red:]"

func UsernameWithColor(username string) string {
	return UsernameColor(username) + username
}

func UsernameColor(username string) string {
	user, err := user.Lookup(username)
	if err != nil {
		return userColor
	}

	if user.Uid == "0" {
		return rootColor
	}

	return userColor
}

func GroupnameWithColor(groupname string) string {
	return GroupnameColor(groupname) + groupname
}

func GroupnameColor(groupname string) string {
	group, err := user.LookupGroup(groupname)
	if err != nil {
		return userColor
	}

	if group.Gid == "0" {
		return rootColor
	}

	return userColor
}
