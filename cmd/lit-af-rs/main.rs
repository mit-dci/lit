#[macro_use] extern crate clap;

extern crate reqwest;

extern crate serde;
#[macro_use] extern crate serde_derive;
extern crate serde_json;

mod litrpc;

fn main() {

    let matches = clap_app!(lit_af_rs =>
        (version: "0.1.0")
        (author: "Trey Del Bonis <j.delbonis.3@gmail.com>")
        (about: "CLI client for Lit")
        (@arg a: +takes_value "Address to connect to.  Default: localhost")
        (@arg p: +takes_value "Port to connect to lit to.  Default: idk yet lmao")
    ).get_matches(); // TODO Make these optional.

    let addr = matches.value_of("a").unwrap_or("localhost");
    let port = match matches.value_of("p").map(str::parse) {
        Some(Ok(p)) => p,
        Some(Err(_)) => panic!("port is not a number"),
        None => 12345 // FIXME
    };

    let _client = litrpc::LitRpcClient::new(addr, port);

    println!("addr: {}, port {}", addr, port);

}
