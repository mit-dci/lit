#[macro_use] extern crate clap;
extern crate cursive;
extern crate reqwest;
extern crate serde;
#[macro_use] extern crate serde_derive;
extern crate serde_json;

use std::panic;

use cursive::Cursive;
use cursive::direction::*;
use cursive::event::*;
use cursive::theme::*;
use cursive::view::*;
use cursive::views::*;

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

    println!("addr: {}, port {}", addr, port);

    match panic::catch_unwind(|| run_ui(addr, port)) {
        Ok(_) => {}, // we're ok
        Err(_) => run_bsod()
    }

}

fn run_ui(addr: &str, port: u16) {

    let mut client = litrpc::LitRpcClient::new(addr, port);

    let mut layout = LinearLayout::new(Orientation::Horizontal);

    let mut c_view = LinearLayout::new(Orientation::Vertical);
    c_view.add_child(TextView::new("Channels"));
    layout.add_child(IdView::new("chans", Panel::new(c_view)));

    let mut txo_view = LinearLayout::new(Orientation::Vertical);
    txo_view.add_child(TextView::new("Txos"));
    layout.add_child(IdView::new("txos", Panel::new(txo_view)));

    let mut siv = Cursive::new();
    siv.add_layer(BoxView::new(SizeConstraint::Full, SizeConstraint::Full, layout));

    siv.set_theme(load_theme(include_str!("ncurses_theme.toml")).unwrap());
    siv.add_global_callback(Event::Refresh, make_update_ui_callback_with_client(&mut client));
    siv.set_fps(1);

    siv.run()

}

fn run_bsod() {

    let mut siv = Cursive::new();

    let d = Dialog::around(TextView::new("RS-AF has encountered an error and needs to exit."))
                    .title("Panic")
                    .button("Exit", |s| s.quit());

    siv.add_layer(d);
    siv.run();

}

fn generate_view_for_chan(chan: litrpc::ChanInfo) -> impl View {

    let mut data = LinearLayout::new(Orientation::Vertical);
    data.add_child(TextView::new(format!("Channel # {}", chan.CIdx)));
    data.add_child(TextView::new(format!("Outpoint: {}", chan.OutPoint)));
    data.add_child(TextView::new(format!("Peer: {}", chan.PeerIdx)));
    data.add_child(TextView::new(format!("Coin Type: {}", chan.CoinType)));
    data.add_child(DummyView);

    data.add_child(TextView::new(format!("Balance: {}/{}", chan.MyBalance, chan.Capacity)));
    let mut bar = ProgressBar::new().range(0, chan.Capacity as usize);
    bar.set_value(chan.MyBalance as usize);
    data.add_child(bar);

    let cbox = BoxView::new(SizeConstraint::Full, SizeConstraint::AtLeast(5), data);
    Panel::new(cbox)

}

fn generate_view_for_txo(txo: litrpc::TxoInfo) -> impl View {

    let strs = vec![
        ("Outpoint", txo.OutPoint),
        ("Amount", format!("{}", txo.Amt)),
        ("Height", format!("{}", txo.Height)),
        ("Coin Type", txo.CoinType)
    ];

    let mut data = LinearLayout::new(Orientation::Vertical);
    for (k, v) in strs {
        data.add_child(TextView::new(format!("{}: {}", k, v)));
    }

    let cbox = BoxView::new(SizeConstraint::Full, SizeConstraint::AtLeast(5), data);
    Panel::new(cbox)

}

fn make_update_ui_callback_with_client(cl: &mut litrpc::LitRpcClient) -> impl Fn(&mut Cursive) {

    use std::mem;

    eprintln!("foo");

    let p: *mut litrpc::LitRpcClient = unsafe { mem::transmute(cl) };

    move |c: &mut Cursive| {

        eprintln!("bar");

        // Channels.
        c.call_on_id("chans", |cpan: &mut Panel<LinearLayout>| {

            eprintln!("baz1");

            let r: &mut litrpc::LitRpcClient = unsafe { mem::transmute(p) };

            eprintln!("baz2");

            let chans: Vec<litrpc::ChanInfo> = match r.call_chan_list(0) {
                Ok(clr) => clr.Channels,
                Err(err) => panic!("{:?}", err)
            };

            let mut c_view = LinearLayout::new(Orientation::Vertical);
            c_view.add_child(TextView::new("Channels"));
            chans.into_iter()
                .map(generate_view_for_chan)
                .for_each(|e| c_view.add_child(e));

            *cpan.get_inner_mut() = c_view;

            eprintln!("baz3");

        });

        // Txos
        c.call_on_id("txos", |txopan: &mut Panel<LinearLayout>| {

            eprintln!("quux1");

            let r: &mut litrpc::LitRpcClient = unsafe { mem::transmute(p) };

            eprintln!("quux2");

            let txos: Vec<litrpc::TxoInfo> = match r.call_get_txo_list() {
                Ok(txr) => txr.Txos,
                Err(err) => panic!("{:?}", err)
            };

            let mut txo_view = LinearLayout::new(Orientation::Vertical);
            txo_view.add_child(TextView::new("Txos"));
            txos.into_iter()
                .map(generate_view_for_txo)
                .for_each(|e| txo_view.add_child(e));

            *txopan.get_inner_mut() = txo_view;

            eprintln!("quux3");

        });

    }
}
