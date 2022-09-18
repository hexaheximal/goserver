# goserver
A Minecraft Classic server, written in [Go](https://go.dev/).

# Feature Comparison

Features | goserver | Original Server Software
|:--|:--|:--
| Language | Go | Java
| Storage | 3MB binary | 200MB OpenJDK + 60KB jar
| Threads | Main thread, Level save thread, and 1 thread for each connection | **A LOT**
| Game updates | Asynchronous | Tick system


# How do I run this on [OS]?
## Linux
Install [Go](https://go.dev/), clone this repository with git, and then run `go build`.
## MacOS
Use the Linux instructions.
## BSD
Use the Linux instructions.
## Windows
you don't.
