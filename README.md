# Synapse_manager
Help

```
  -deactivate
    	Deactivate an account, requires --target
  -list
    	List all users, requires no arguments
  -query
    	Queries a user and gets its current information, needs --target
  -reset
    	Reset users account with new password, needs --target and --pass
  -target string
    	The user account to be acted upon (if required)
  -url string
    	The URL that points towards the matrix server (default "http://localhost:8008")
```

On taking any of these actions an admin username and password will be needed to be entried via stdin. 

This will then be used to connect to the synapse server to get an authorization token to thus perform the action. 