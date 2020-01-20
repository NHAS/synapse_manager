# Synapse_manager
Help

```
  -deactivate string
    	Deactivate an account, eg -deactivate @target:matrix.ais
  -list
    	List all users, requires no arguments
  -query string
    	Queries a user and gets last ip, user agent, eg -query @target:matrix.ais
  -reset string
    	Reset users account with new password, eg -reset @target:matrix.ais
  -url string
    	The URL that points towards the matrix server (default "http://localhost:8008")
```

On taking any of these actions an admin username and password will be needed to be entried via stdin. 

This will then be used to connect to the synapse server to get an authorization token to thus perform the action. 