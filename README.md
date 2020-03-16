# Synapse_manager
Help

```
Usage of ./synapse_manager:
  -autopurge
        Purge all rooms with 0 members joined to them
  -deactivate string
        Deactivate an account, eg -deactivate @target:matrix.ais
  -list
        List all users, requires no arguments
  -list_rooms
        List all rooms, requires no arguments
  -purge string
        Purge a room from the database, typically so it can be reclaimed if everyone left, eg -purge !oqhoCmLzNgkVlLgxQp:matrix.ais, this can be found in the database of room_aliases
  -query string
        Queries a user and gets last ip, user agent, eg -query @target:matrix.ais
  -reset string
        Reset users account with new password, eg -reset @target:matrix.ais
  -url string
        The URL that points towards the matrix server (default "http://localhost:8008")
```
On taking any of these actions an admin username and password will be needed to be entried via stdin. 

This will then be used to connect to the synapse server to get an authorization token to thus perform the action. 

While the purge API has now been fixed, some rooms that have been created have users joined to them, which stops a room from being purged. There is an API to shutdown a room and kick all people, however this creates a new room. Which.... kind of defeats the purpose of things to begin with. 
