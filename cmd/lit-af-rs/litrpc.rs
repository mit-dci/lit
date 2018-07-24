#![allow(unused)]
#![allow(non_snake_case)]

#![feature(concat_idents)]

use std::net::TcpStream;

use std::fmt;

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
    //#[serde(default="Option::None")]
    result: Option<R>,

    //#[serde(default="Option::None")]
    error: Option<RpcError>,

    id: u64
}

#[derive(Clone, Debug, Deserialize)]
pub struct RpcError {
    code: i64,
    message: String,
    data: String
}

pub struct LitRpcClient {
    next_msg_id: u64,
    url: String,
    client: reqwest::Client
}

#[derive(Debug)]
pub enum LitRpcError {
    RpcError(RpcError),
    RpcInvalidResponse,
    SerdeJsonError(serde_json::Error),
    HttpError(reqwest::Error),
}

impl From<reqwest::Error> for LitRpcError {
    fn from(from: reqwest::Error) -> Self {
        LitRpcError::HttpError(from)
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
        println!("request: {}", req_body);

        // Send it off and get a response.
        let mut res_json = self.client.post(self.url.as_str())
            .body(req_body)
            .send()?;
        let text = res_json.text()?;
        println!("reponse: {}", text);

        // Deserialize...
        let res: RpcResponse<R> = serde_json::from_str(text.as_ref())?;
        if res.id != req.id {
            // Ok this makes no sense but we should fail out anyways.
            return Err(LitRpcError::RpcInvalidResponse)
        }

        // And then match on the response part thingies to figure out what to do.
        match (res.result, res.error) {
            (Some(r), None) => Ok(r),
            (None, Some(e)) => Err(LitRpcError::RpcError(e)),
            _ => Err(LitRpcError::RpcInvalidResponse)
        }

    }
}

macro_rules! rpc_call {
    {$fname:ident, $method:ident, { $($ii:ident: $it:ty),* } => $oname:ident { $($oi:ident: $ot:ty),* }} => {

        #[derive(Clone, Deserialize)]
        #[allow(non_snake_case)]
        pub struct $oname {
            $( pub $oi: $ot ),*
        }

        impl LitRpcClient {

            #[allow(non_snake_case)]
            pub fn $fname(&mut self, $($ii: $it),*) -> Result<$oname, LitRpcError> {

                #[derive(Clone, Debug, Serialize)]
                #[allow(non_snake_case)]
                struct Params {
                    $( $ii: $it ),*
                };

                let arg = Params {
                    $($ii: $ii),*
                };

                self.call(concat!("LitRPC.", stringify!($method)), arg)
            }
        }

    };
    {$fname:ident, $method:ident, { $($ii:ident: $it:ty),* } => $oname:ident} => {

        impl LitRpcClient {

            #[allow(non_snake_case)]
            pub fn $fname(&mut self, $($ii: $it),*) -> Result<$oname, LitRpcError> {

                #[derive(Clone, Debug, Serialize)]
                #[allow(non_snake_case)]
                struct Params {
                    $( $ii: $it ),*
                };

                let arg = Params {
                    $($ii: $ii),*
                };

                self.call(concat!("LitRPC.", stringify!($method)), arg)
            }
        }

    };
}

#[derive(Clone, Deserialize)]
pub struct StatusReply {
    Status: String
}

impl fmt::Debug for StatusReply {
    fn fmt(&self, f: &mut fmt::Formatter) -> Result<(), fmt::Error> {
        f.write_str(format!("status: {}", self.Status).as_ref())
    }
}

// netcmds

#[derive(Clone, Deserialize)]
pub struct ListeningPortsReply {
    LisIpPorts: Vec<String>,
    Adr: String
}

rpc_call! {
    call_lis, Listen,
    { Port: String } => ListeningPortsReply
}

rpc_call! {
    call_connect, Connect,
    { LNAddr: String } => StatusReply
}

rpc_call! {
    call_assign_nickname, AssignNickname,
    { Peer: u32, Nickname: String } => StatusReply
}

#[derive(Clone, Deserialize)]
pub struct ConInfo {
    PeerNumber: u32,
    RemoteHost: String
}

rpc_call! {
    call_list_connections, ListConnections,
    {} => ListConnectionsReply { Connections: Vec<ConInfo>, MyPKH: String }
}

rpc_call! {
    call_get_listening_ports, GetListeningPorts,
    {} => ListeningPortsReply
}

rpc_call! { call_get_messages, GetMessages, {} => StatusReply }
rpc_call! { call_say, Say, {} => StatusReply }
rpc_call! { call_stop, Stop, {} => StatusReply }

rpc_call! {
    call_get_chan_map, GetChannelMap,
    {} => ChanGraphReply { Graph: String }
}

// chancmds

#[derive(Clone, Debug, Deserialize)]
pub struct ChanInfo {
    pub OutPoint: String,
    pub CoinType: u32,
    pub Closed: bool,
    pub Capacity: i64,
    pub MyBalance: i64,
    pub Height: i32,
    pub StateNum: i64,
    pub PeerIdx: u32,
    pub CIdx: u32,
    pub Data: [u8; 32],
    pub Pkh: [u8; 20]
}

rpc_call! {
    call_chan_list, ChannelList,
    { ChanIdx: u32 } => ChanListReply { Channels: Vec<ChanInfo> }
}

rpc_call! {
    call_fund, FundChannel,
    {
        Peer: u32,
        CoinType: u32,
        Capacity: i64,
        Roundup: i64,
        InitialSend: i64,
        Data: [u8; 32]
    } => StatusReply
}

rpc_call! {
    call_dual_fund, DualFundChannel,
    {
        Peer: i32,
        CoinType: i32,
        OurAmount: i64,
        TheirAmount: i64
    } => StatusReply
}

// TODO DualFundDecline, DualFundAccept, etc.

rpc_call! {
    call_get_pending_dual_fund, PendingDualFund,
    {} => PendingDualFundReply {
        Pending: bool,
        PeerIdx: u32,
        CoinType: u32,
        TheirAmount: i64,
        RequestedAmount: i64
    }
}

/* TODO No auto-Deserialize for [u8; 64], do something about this later.
#[derive(Clone, Deserialize)]
pub struct JusticeTx {
    Sig: [u8; 64],
    Txid: [u8; 16],
    Amt: i64,
    Data: [u8; 32],
    Pkh: [u8; 20],
    Idx: u64
}

rpc_call! {
    call_get_state_dump, StateDump,
    {} => StateDumpReply {
        Txs: Vec<JusticeTx>
    }
}
*/

rpc_call! {
    call_push, Push,
    {
        ChanIdx: u32,
        Amt: i64,
        Data: [u8; 32]
    } => PushReply {
        StateIdx: u64
    }
}

rpc_call! {
    call_close_chan, CloseChannel,
    { ChanIdx: u32 } => StatusReply
}

rpc_call! {
    call_break_chan, BreakChannel,
    { ChanIdx: u32 } => StatusReply
}

#[derive(Clone, Deserialize)]
pub struct PrivInfo {
    pub OutPoint: String,
    pub Amt: i64,
    pub Height: i32,
    pub Delay: i32,
    pub CoinType: String,
    pub Witty: bool,
    pub PairKey: String,
    pub WIF: String
}

rpc_call! {
    call_dump_privs, DumpPrivs,
    {} => DumpPrivsReply { Privs: Vec<PrivInfo> }
}

// walletcmds

#[derive(Clone, Deserialize)]
pub struct CoinBalInfo {
    pub CoinType: u32,
    pub SyncHeight: i32,
    pub ChanTotal: i64,
    pub TxoTotal: i64,
    pub MatureWitty: i64,
    pub FeeRate: i64
}

rpc_call! {
    call_bal, Balance,
    {} => BalanceReply { Balances: Vec<CoinBalInfo> }
}

#[derive(Clone, Debug, Deserialize)]
pub struct TxoInfo {
    pub OutPoint: String,
    pub Amt: i64,
    pub Height: i32,
    pub Delay: i32,
    pub CoinType: String,
    pub Witty: bool,
    pub KeyPath: String
}

rpc_call! {
    call_get_txo_list, TxoList,
    {} => TxoListReply { Txos: Vec<TxoInfo> }
}

#[derive(Clone, Deserialize)]
pub struct TxidsReply {
    pub Txids: Vec<String>
}

rpc_call! {
    call_send, Send, // oh no it's already an identifier whoops
    {
        DestAddrs: Vec<String>,
        Amts: Vec<i64>
    } => TxidsReply
}

rpc_call! {
    call_sweep, Sweep,
    {
        DestAdr: String,
        NumTx: u32,
        Drop: bool // oh no it's an identifier too, whoops!
    } => TxidsReply
}

rpc_call! {
    call_fanout, Fanout,
    {
        DestAdr: String,
        NumOutputs: u32,
        AmtPerOutput: i64
    } => TxidsReply
}

#[derive(Clone, Deserialize)]
pub struct FeeReply {
    pub CurrentFee: i64
}

rpc_call! {
    call_set_fee, SetFee,
    { Fee: i64, CoinType: u32 } => FeeReply
}

rpc_call! {
    call_get_fee, GetFee,
    { CoinType: u32 } => FeeReply
}

rpc_call! {
    call_gen_address, Address,
    {
        NumToMake: u32,
        CoinType: u32
    } => AddressReply {
        WitAddresses: Vec<String>,
        LegacyAddresses: Vec<String>
    }
}

// TODO make RPC call definitions
