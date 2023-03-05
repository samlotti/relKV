# relKV

A key value data store exposed over http(s).

Features:

- relationship support (via segments)
- alias support (alternate indexes)
- backup with scp embedded
- status page

To start the server first copy the env.template and set the various attributes. It expects a data directory and backup directory to be setup. It also will read the .env file so environment variables to can configured in the enviroment or via that file.

# Http commands

- Get /status  
  Shows the server status. Will return 500 if there are issues. Can be pinged using a monitoring system to alter of issues.
- Put /bucket
  Create a new bucket.
- Get /
  list all buckets.
- Get /bucket  
  search a bucket. Uses header attributes to configurate the search.

  - skip, max <- paging support
  - segments <- :segments: in keyportion, : separated
  - prefix <- limit to prefixes
  - values <- t/f default is false
  - b64 <- return values as base64
  - explain <- dont return data, return headers showing how many rows were read for the request.

- Post /bucket
  Returns the values of the keys in a batch
  The body to send is a list of keys, each on a separate line.
  Headers:

  - b64 <- return values as base64

- Post /bucket/key
  Insert or update a key.
  The body is the content.
  Headers:

  - aliases <- The alternate index values ; separated

- Get /bucket/key
  Returns a single key value as the content.

- DELETE /bucket/key
  Delete the key
  Headers:
  - aliases <- The alternate index values ; separated

# Segments

Segments are parts of keys separated by :

This allows keys to be searched with a filtering on parts of the keys.
Note that designing keys and alternates to be filtered in sort order provides a very efficient means to access
rows in the kv store.

For example if you are storing a games between 2 players then you may want an alternate key with this format:

p1Id:p2Id:{rated/unrated}:gameId
p2Id:p1Id:{rated/unrated}:gameId
with a primary key as:
gameId

A very fast query to find rated games between 2 players would be a get with 'prefix' of p1Id:p2Id or p2Id:p1Id and a 'segments' of 'rated'

# Aliases

Aliases allow an alternate index to be created. These become keys added like other keys but point to the original
data. It's up to the caller to set the alias for the key during the create call. Updating the primary key will
also update the alias since it is a pointer to the key. Deleting the primary will require the aliases to be placed in
the header in order to maintain a valid structure.

Example usage: If you have a game with 2 players and want to be able to find the game or games for the players can do:
Add key gameId1
alias=player1:player2:gameId1;player2:player1:gameId1

This allows prefix searched by players, also with segments can find games where both players played. (use prefix and segment options)

Structure in the KV Store:
gameId1 -> { the json game data }
player1:player2:gameId1 -> gameId1
player2:player1:gameId1 -> gameId1

So there are 3 keys stored.

- Care must be taken that aliases are unique, that is why the final segment is the gameId

This structure allows a simple get for the game Id
Searching for games played by players and
games played by both players.
Prefix search by first node provides efficient lookup.

Note the kv stored doesn't maintain the relationship from key to aliases, its up to the application
to maintain this relationship.

Orphaned aliases with not show in search results. They will be translarent to the caller but will take up some space in the kv store.

Duplicate keys for aliases.
It is possible that on the creation of a new key or the update of a key that a duplicate can exist.

If during write a duplicate is detected the response will be:
and an error returned headerkey -> duplicate_key value is the alias key.

# Notes

For calls that require a bucket, if it is not found then StatusBadRequest is returned.
This is to differentiate between key not found -> StatusNotFound

The getKeys will return error entries in this case since the alias was explicitly specified

# Backups

The store can be conigured to run a backup on certain hours and then optionally scp them to another server.

# Restore

To restore a backup do the following.

copy the zip or backup file to a folder

in the current relKv directory

./relKv stop

./relKv restore {name of backup file} {name of database for restore}

Note that you cannot restore to an existing database so delete that database directory.
The backup file can be a zip file from the backup. It can only contain a single file.

Can restore to any name of a database. This makes it easy to restore it, try it and then just rename the directory.

# Start the kv store

./relKv

Run this in a directory that contains a .env file or set the environment variables

# Stopping the kv store

./relKv stop

Run this in a directory that contains a .env file or set the environment variables

# Environment variables

See the .env.template

# Security

At this time the code uses a token for accessing the http endpoint. (/status is not secured). It is recommended to run behind a reverse proxy with https enables.

The token is defined in the .env file, make sure not to check this into source control.

The key in the .env is:
SECRET=some long random number.

The header key is 'tkn'.

If not token is defined at server start then it will not be needed to access the http endoints.

# Dependencies

This project uses badger (https://github.com/dgraph-io/badger) for the backing store.
