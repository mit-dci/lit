This project was bootstrapped with [Create React App](https://github.com/facebookincubator/create-react-app).

## About Lit Webui

This is a simple web based UI for lit, originally written by Joe Chung

At the moment (April 2018) it was built on two main libraries: React (using the Create React App starter project)
and Material-UI and I've purposely avoided Redux and other libraries for the purposes of simplicity

To run the UI, you must be on the same machine as your local lit node. Just cd to the webui directory and do npm start.
That should throw open a browser window and the UI which will connect via websockets RPC to the local node on port 8001.
