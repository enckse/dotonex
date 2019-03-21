mod useradd;

mod constants {
    pub const CONFIG_DIR: &str = "config";
}

use crate::useradd::{get_pass, new_user};
use std::env;

fn main() {
    let args: Vec<_> = env::args().collect();
    if args.len() < 2 {
        println!("no command given");
        return;
    }
    let mut client = false;
    let mut pass: String = String::new();
    let mut user: String = String::new();
    let command = args[1].to_string();
    if args.len() > 2 {
        let mut idx = -1;
        for a in args.into_iter() {
            idx += 1;
            if idx < 2 {
                continue;
            }
            let parts: Vec<&str> = a.split("=").collect();
            if parts.len() != 2 {
                println!("invalid parameter input");
                return;
            }
            match parts[0] {
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
        _ => {
            println!("command unknown: {}", command);
        }
    }
    if !valid {
        std::process::exit(1);
    }
}
