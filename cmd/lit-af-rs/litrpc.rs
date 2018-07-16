#![allow(unused)]

use std::net::TcpStream;

use serde::{Serialize, Deserialize};
use serde::de::DeserializeOwned;

use serde_json;

use websocket::client::sync;
use websocket::ClientBuilder;
use websocket::message;

#[derive(Clone, Debug, Serialize)]
struct RpcReqest<P> where P: Serialize {
    method: String,
    params: P,
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
    msg_id: u64,
    client: sync::Client<TcpStream>
}

impl<'de> LitRpcClient {
    pub fn new(addr: &str, port: u16) -> Result<LitRpcClient, ()> {
        let url = format!("ws://{}:{}/ws", addr, port);
        let mut builder = ClientBuilder::new(url.as_ref()).unwrap();

        let client = builder.connect_insecure().unwrap(); // FIXME Error handling.

        Ok(LitRpcClient {
            msg_id: 0,
            client: client
        })
    }

    pub fn call<P: Serialize, R: DeserializeOwned>(&mut self, name: &str, params: P) -> Result<R, ()> {

        // Construct the request object.
        let req = RpcReqest {
            method: String::from(name),
            params: params,
            jsonrpc: String::from("2.0"), // required by the standard
            id: self.msg_id
        };

        self.msg_id += 1;

        // Serialize and send.
        let req_json = serde_json::to_string(&req).unwrap();
        self.client.send_message(&message::OwnedMessage::Text(req_json)).unwrap();

        // Wait for response and return.
        let mut tries = 10;
        while tries > 10 {

            let resp_raw = self.client.recv_message().unwrap();

            if let message::OwnedMessage::Text(s) = resp_raw {
                return Ok(serde_json::from_str(s.as_ref()).unwrap());
            }

            tries -= 1;

        }

        Err(()) // timed out

    }
}
