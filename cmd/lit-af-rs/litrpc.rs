#![allow(unused)]

use std::net::TcpStream;

use reqwest;

use serde::{Serialize, Deserialize};
use serde::de::DeserializeOwned;

use serde_json;

#[derive(Clone, Debug, Serialize)]
struct RpcReqest<P> where P: Serialize {
    method: String,
    params: Vec<P>,
    jsonrpc: String,
    id: u64
}

#[derive(Clone, Debug, Deserialize)]
struct RpcResponse<R> {
    jsonrpc: String,

    //#[serde(default="Option::None")]
    result: Option<R>,

    //#[serde(default="Option::None")]
    error: Option<RpcError>,

    id: u64
}

#[derive(Clone, Debug, Deserialize)]
struct RpcError {
    code: i64,
    message: String,
    data: String
}

pub struct LitRpcClient {
    next_msg_id: u64,
    url: String,
    client: reqwest::Client
}

pub enum LitRpcError {
    SerdeJsonError(serde_json::Error),
    ReqError(reqwest::Error),
    UnknownError
}

impl From<reqwest::Error> for LitRpcError {
    fn from(from: reqwest::Error) -> Self {
        LitRpcError::ReqError(from)
    }
}

impl From<serde_json::Error> for LitRpcError {
    fn from(from: serde_json::Error) -> Self {
        LitRpcError::SerdeJsonError(from)
    }
}

impl LitRpcClient {
    pub fn new(addr: &str, port: u16) -> LitRpcClient {
        LitRpcClient {
            next_msg_id: 0,
            url: format!("http://{}:{}/oneoff", addr, port),
            client: reqwest::Client::new()
        }
    }

    pub fn call<P: Serialize, R: DeserializeOwned>(&mut self, name: &str, params: P) -> Result<R, LitRpcError> {

        // Construct the request object.
        let req = RpcReqest {
            method: String::from(name),
            params: vec![params],
            jsonrpc: String::from("2.0"), // required by the standard
            id: self.next_msg_id
        };

        // Increment the "next" value to not confuse request IDs.
        self.next_msg_id += 1;

        // Serialize the request.
        let req_body = serde_json::to_string(&req)?;

        // Send it off and get a response.
        let mut res_json = self.client.post(self.url.as_str())
            .body(req_body)
            .send()?;

        // Just deserialize.
        Ok(serde_json::from_str(res_json.text()?.as_ref())?)

    }
}
