# Synapse_manager
Help

```
  -deactivate string
        Deactivate an account, eg -deactivate @target:matrix.ais
  -list
        List all users, requires no arguments
  -purge string
        Purge a room from the database, typically so it can be reclaimed if everyone left, eg -purge !QjnfHHjcIUOrqSOJib:matrix.ais, this can be found in the database of room_aliases
  -query string
        Queries a user and gets last ip, user agent, eg -query @target:matrix.ais
  -reset string
        Reset users account with new password, eg -reset @target:matrix.ais
  -url string
        The URL that points towards the matrix server (default "http://localhost:8008")
```

On taking any of these actions an admin username and password will be needed to be entried via stdin. 

This will then be used to connect to the synapse server to get an authorization token to thus perform the action. 

Purge will currently return an error, however it frees the room to be reallocated. So just a bit broken in synapse, as of 5 days ago, there is a patch to fix that, and will be in the next version (1.9.0)