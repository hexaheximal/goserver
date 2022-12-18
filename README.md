# goserver
A Minecraft Classic server, written in [Go](https://go.dev/).

# ⚠️ WARNING ⚠️

**DO NOT USE THIS BRANCH UNLESS YOU KNOW WHAT YOU'RE DOING. USE THE STABLE BRANCH INSTEAD.**  

Major changes are going to be made:
- [x] Splitting the code in `protocol.go` (and some of the code in `main.go`) into separate files
- [x] Rewriting the protocol & serialization system
- [ ] Implementing admin commands
- [ ] Implementing a plugin system
- [ ] Implementing multiple levels & switching between them

# Feature Comparison

Features | goserver | Original Server Software
|:--|:--|:--
| Language | Go | Java
| Storage | 3MB binary | 200MB OpenJDK + 60KB jar
| Threads | Main thread, Level save thread, and 1 thread for each connection | **A LOT**
| Game updates | Asynchronous | Tick system

TL;DR goserver is better

# How do I run this on [OS]?
## Linux
Install [Go](https://go.dev/), clone this repository with git, and then run `go build`.
## MacOS
Use the Linux instructions.
## BSD
Use the Linux instructions.
## Windows
you don't.
