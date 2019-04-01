mod configure;
mod encrypt;
mod objects;
mod useradd;
mod constants {
    use rand::distributions::Alphanumeric;
    use rand::Rng;
    pub const CONFIG_DIR: &str = "config";
    pub const PASSWORDS: &str = "passwords";
    pub const OUTPUT_DIR: &str = "bin/";
    pub const MANIFEST: &str = "manifest";
    pub const EAP_USERS: &str = "eap_users";
    pub const IS_YAML: &str = ".yaml";
    pub fn random_string(length: usize) -> String {
        return rand::thread_rng()
            .sample_iter(&Alphanumeric)
            .take(length)
            .collect::<String>();
    }
}

use crate::configure::{all, netconf};
use crate::constants::CONFIG_DIR;
use crate::encrypt::{decrypt_file, encrypt_file};
use crate::useradd::{get_pass, new_user};
use std::env;
use std::path::Path;

fn main() {
    let args: Vec<_> = env::args().collect();
    if args.len() < 2 {
        println!("no command given");
        return;
    }
    let mut client = false;
    let mut pass = String::new();
    let mut user = String::new();
    let command = args[1].to_string();
    if !Path::new(CONFIG_DIR).exists() {
        println!("config directory missing...");
        return;
    }
    if args.len() > 2 {
        let mut idx = -1;
        for a in args.into_iter() {
            idx += 1;
            if idx < 2 {
                continue;
            }
            if !a.starts_with("--") {
                println!("parameter must start with --");
                return;
            }
            let parts: Vec<&str> = a.split("=").collect();
            if parts.len() != 2 {
                println!("invalid parameter input");
                return;
            }
            let mut p = String::new();
            p.push_str(&parts[0][2..]);
            match &*p {
                "user" => {
                    user = parts[1].to_string();
                }
                "pass" => {
                    pass = parts[1].to_string();
                }
                "client" => {
                    client = parts[1] == "true";
                }
                _ => println!("unknown parameter: {}", parts[0]),
            }
        }
    }
    let mut valid = false;
    let cmd: &str = &*command;
    match cmd {
        "useradd" => {
            valid = new_user(&user, &pass);
        }
        "pwd" => {
            valid = get_pass(&pass);
        }
        "enc" => {
            valid = encrypt_file(&pass);
        }
        "dec" => {
            valid = decrypt_file(&pass);
        }
        "configure" => {
            valid = all(client);
        }
        "netconf" => {
            valid = netconf();
        }
        _ => {
            println!("command unknown: {}", command);
        }
    }
    if !valid {
        std::process::exit(1);
    }
}
