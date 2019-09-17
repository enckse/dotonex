mod configure;
mod encrypt;
mod useradd;
mod constants {
    use rand::distributions::Alphanumeric;
    use rand::Rng;
    pub const CONFIG_DIR: &str = "config";
    pub const PASSWORDS: &str = "passwords";
    pub const OUTPUT_DIR: &str = "bin/";
    pub const MANIFEST: &str = "manifest";
    pub const EAP_USERS: &str = "eap_users";
    pub fn random_string(length: usize) -> String {
        rand::thread_rng()
            .sample_iter(&Alphanumeric)
            .take(length)
            .collect::<String>()
    }
}

extern crate clap;
use crate::configure::{all, netconf};
use crate::constants::CONFIG_DIR;
use crate::encrypt::{decrypt_file, encrypt_file};
use crate::useradd::{get_pass, new_user, passwd};
use std::env;
use std::path::Path;
use clap::{App, Arg};

fn main() {
    let matches = App::new("radiucal-admin")
        .arg(
            Arg::with_name("command")
                .help("command to perform")
                .takes_value(true),
        )
        .arg(
            Arg::with_name("server")
                .short("s")
                .long("server")
                .help("operate in server-mode")
        )
        .arg(
            Arg::with_name("pass")
                .short("p")
                .long("pass")
                .help("administrative password")
                .takes_value(true)
        ).get_matches();
    if !Path::new(CONFIG_DIR).exists() {
        println!("config directory missing...");
        return;
    }
    let cmd = matches.value_of("command").expect("no command given...");
    let server = matches.is_present("server");
    let mut pass = String::from(matches.value_of("pass").unwrap_or(""));
    if pass.is_empty() {
        if let Ok(v) =  env::var("RADIUCAL_ADMIN_KEY") {
            pass = v;
        }
    }
    let mut valid = false;
    match cmd {
        "useradd" => {
            valid = new_user(&pass);
        }
        "pwgen" => {
            valid = get_pass();
        }
        "passwd" => {
            valid = passwd(&pass);
        }
        "enc" => {
            valid = encrypt_file(&pass);
        }
        "dec" => {
            valid = decrypt_file(&pass);
        }
        "configure" => {
            valid = all(server);
        }
        "netconf" => {
            valid = netconf();
        }
        _ => {
            println!("command unknown: {}", cmd);
        }
    }
    if !valid {
        std::process::exit(1);
    }
}
