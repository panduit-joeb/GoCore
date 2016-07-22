#GoCore Application Settings

There are 2 components to GoCore which must be configured within your application:

##buildCore

Create a build package for your application with the following:

	package main

	import (
		"github.com/DanielRenne/GoCore/buildCore"
	)
	
	func main() {
		buildCore.Initialize("src/github.com/DanielRenne/GoCoreHelloWorld")
	}

##app

The GoCore/core/app package is what runs your application.  You must first Initialize() it with the root path of your application.  Then call the Run() method.
	
	package main
	
	import (
		"github.com/DanielRenne/GoCore/core/app"
		_ "github.com/DanielRenne/GoCoreHelloWorld/webAPIs/v1/webAPI"
	)
	
	func main() {
		//Run First.
		app.Initialize("src/github.com/DanielRenne/GoCoreHelloWorld")
	
		//Add your Application Code here.
	
		//Run Last.
		app.Run()
	}

#App Settings

GoCore reads a file located in the root directory of your package called WebConfig.json  If one does not exist `buildCore` will auto generate one for your package with default settings.

##WebConfig.json

There are two root objects to be configured:

###application



	"application":{
	    "domain": "127.0.0.1",
	    "httpPort": 80,
	    "httpsPort": 443, 
	    "releaseMode":"release",
	    "webServiceOnly":false,
	    "info":{
	    	"title": "Hello World Playground",
	    	"description":"A web site to try GoCore.",
	    	"contact":{
	    		"name":"DRenne",
	    		"email":"support@myWebSite.com",
	    		"url":"myWebSite.com"
	    	},
	    	"license": {
	    		"name": "Apache 2.0",
	  			"url": "http://www.apache.org/licenses/LICENSE-2.0.html"
	    	},
	    	"termsOfService":"http://127.0.0.1/terms"
	    },
		"htmlTemplates":{
			"enabled":false,
			"directory":"templates",
			"directoryLevels": 1
		}
	}

At the root of application there are the following fields:

####domain

Tells the application which domain to redirect https traffic to.

####httpPort, httpsPort

Tells the application which ports to listen on for http and https.

####releaseMode

Tells the application to debug and run GIN http routing into release mode.  "release" will enable release.  An empty string will place the application in debug mode.

####webServiceOnly

Tells the application only route web service paths.  NO static file routing will be enabled when set to true.

####info

Tells the application details about the application for swagger.io information and schema.

####htmlTemplates

Tells the application to use HTML templates that conform to the GIN Engine.  See [HTML Rendering in GIN](https://github.com/gin-gonic/gin#html-rendering]).  See [HTML Templates](https://github.com/DanielRenne/GoCore/blob/master/doc/HTML_Templates.md) for more details and examples.


###dbConnections

Provides an array of database connections.  Currently GoCore only supports a single database connection.  Future releases will allow for multiple connections and types.

	"dbConnections":[
		{
			"driver" : "boltDB",
			"connectionString" : "db/helloWorld.db"
		}
	]
###Database Connection Examples

###Bolt DB

A NOSQL GOLang native database that runs within your application

		{
			"driver" : "boltDB",
			"connectionString" : "db/helloWorld.db"
		}

###Mongo DB

A NOSQL database that runs outside your application

		{
			"driver" : "mongoDB",
			"connectionString" : "mongodb://myuser:mypass@localhost:40001,otherhost:40001/mydb"
		}

###SQLite3

A SQL Database instance running within your application

		{
			"driver" : "sqlite3",
			"connectionString" : "db/helloWorld.db"
		}

###MYSQL

A SQL Database instance running external to your application

		{
			"driver" : "mysql",
			"connectionString" : " myUsername:myPassword@/HelloWorld"
		}

###MS SQL Server

A SQL Database instance running external to your application

		{
			"driver" : "mssql",
			"connectionString" : "server=myServerAddress;Database=HelloWorld;user id=myUsername;Password=myPassword;Connection Timeout=3000;"
		}