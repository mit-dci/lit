#[macro_use] extern crate clap;
extern crate cursive;
extern crate reqwest;
extern crate serde;
#[macro_use] extern crate serde_derive;
extern crate serde_json;

use std::cmp;
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

    let mut client = Box::new(litrpc::LitRpcClient::new(addr, port));

    let mut layout = LinearLayout::new(Orientation::Horizontal);

    let mut c_view = LinearLayout::new(Orientation::Vertical);
    c_view.add_child(TextView::new("Channels"));
    layout.add_child(
        BoxView::new(
            SizeConstraint::Full,
            SizeConstraint::Full,
            IdView::new("chans", Panel::new(c_view))));

    let mut right_view = LinearLayout::new(Orientation::Vertical);

    // Balances
    let mut bal_view = LinearLayout::new(Orientation::Vertical);
    bal_view.add_child(TextView::new("Balances"));
    right_view.add_child(
        BoxView::new(
                SizeConstraint::Full,
                SizeConstraint::Full,
                IdView::new("bals", Panel::new(bal_view)))
            .squishable());

    // Txos
    let mut txo_view = LinearLayout::new(Orientation::Vertical);
    txo_view.add_child(TextView::new("Txos"));
    right_view.add_child(
        BoxView::new(
                SizeConstraint::Full,
                SizeConstraint::Full,
                IdView::new("txos", Panel::new(txo_view)))
            .squishable());

    layout.add_child(BoxView::new(SizeConstraint::Full, SizeConstraint::Full, right_view));

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

fn generate_view_for_bal(bal: &litrpc::CoinBalInfo, addrs: Vec<String>) -> impl View {

    let mut data = LinearLayout::new(Orientation::Vertical);

    let grand_total = bal.ChanTotal + bal.TxoTotal;
    let bal_str = format!(
        "  Funds: chans {} + txos {} = total {} (sat)",
        bal.ChanTotal,
        bal.TxoTotal,
        grand_total);

    data.add_child(TextView::new(format!("- Type {} @ height {}", bal.CoinType, bal.SyncHeight)));
    data.add_child(TextView::new(bal_str));
    addrs.into_iter()
        .map(|a| format!("  - {}", a))
        .map(TextView::new)
        .for_each(|l| data.add_child(l));
    data.add_child(DummyView);

    data

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

    let cbox = BoxView::new(SizeConstraint::Full, SizeConstraint::Fixed(5), data);
    Panel::new(cbox)

}

fn make_update_ui_callback_with_client(cl: &mut litrpc::LitRpcClient) -> impl Fn(&mut Cursive) {

    use std::mem;

    let clp: *mut litrpc::LitRpcClient = unsafe { mem::transmute(cl) };

    move |c: &mut Cursive| {

        let clrc: &mut litrpc::LitRpcClient = unsafe { mem::transmute(clp) };

        // Channels.
        let chans: Vec<litrpc::ChanInfo> = match clrc.call_chan_list(0) {
            Ok(clr) => clr.Channels,
            Err(err) => panic!("{:?}", err)
        };

        c.call_on_id("chans", |cpan: &mut Panel<LinearLayout>| {

            let mut c_view = LinearLayout::new(Orientation::Vertical);
            c_view.add_child(TextView::new("Channels"));
            chans.into_iter()
                .map(generate_view_for_chan)
                .for_each(|e| c_view.add_child(e));

            *cpan.get_inner_mut() = c_view;

        });

        // Bals
        let bals: Vec<litrpc::CoinBalInfo> = match clrc.call_bal() {
            Ok(br) => {
                let mut bals = br.Balances;
                bals.sort_by(|a, b| cmp::Ord::cmp(&a.CoinType, &b.CoinType));
                bals
            },
            Err(err) => panic!("{:?}", err)
        };

        // Addrs
        let addrs: Vec<(u32, String)> = match clrc.call_get_addresses() {
            Ok(ar) => {
                let mut addrs: Vec<(u32, String)> = ar.CoinTypes.into_iter()
                    .zip(ar.WitAddresses.into_iter())
                    .collect();
                addrs.sort_by(|a, b| match cmp::Ord::cmp(&a.0, &b.0) {
                    cmp::Ordering::Equal => cmp::Ord::cmp(&a.1, &b.1),
                    o => o
                });
                addrs
            },
            Err(err) => panic!("{:?}", err)
        };

        c.call_on_id("bals", |balpan: &mut Panel<LinearLayout>| {

            let mut bal_view = LinearLayout::new(Orientation::Vertical);
            bal_view.add_child(TextView::new("Balances"));
            bal_view.add_child(DummyView);
            bals.into_iter()
                .map(|b| generate_view_for_bal(
                    &b,
                    addrs.clone().into_iter()
                        .filter(|(t, _)| *t == b.CoinType)
                        .map(|(_, a)| a)
                        .collect()))
                .for_each(|e| bal_view.add_child(e));

            *balpan.get_inner_mut() = bal_view;

        });

        // Txos
        let txos: Vec<litrpc::TxoInfo> = match clrc.call_get_txo_list() {
            Ok(txr) => txr.Txos,
            Err(err) => panic!("{:?}", err)
        };

        c.call_on_id("txos", |txopan: &mut Panel<LinearLayout>| {

            let mut txo_view = LinearLayout::new(Orientation::Vertical);
            txo_view.add_child(TextView::new("Txos"));
            txos.into_iter()
                .map(generate_view_for_txo)
                .for_each(|e| txo_view.add_child(e));

            *txopan.get_inner_mut() = txo_view;

        });

    }
}
