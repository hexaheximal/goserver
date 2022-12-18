# goserver
A Minecraft Classic server, written in [Go](https://go.dev/).

# ⚠️ WARNING ⚠️

TL;DR **DO NOT USE THIS BRANCH UNLESS YOU KNOW WHAT YOU'RE DOING. USE THE STABLE BRANCH INSTEAD.**  

After a long time of not making any changes to goserver, I'm going to make a lot of major changes to it.  
***More specifically:***
- ✅ Splitting the code in `protocol.go` (and some of the code in `main.go`) into separate files
- ☑️ Rewriting the protocol & serialization system
- ❌ Implementing admin commands
- ❌ Implementing a plugin system
- ❌ Implementing multiple levels & switching between them

✅ = Finished  
☑️ = Partially implemented  
❌ = Not implemented yet  

...and probably also some other things that I haven't planned yet.  

If you need stability when using goserver, you should use the `stable` branch until I finish making all of the major changes.  

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
