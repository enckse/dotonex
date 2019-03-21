use crate::constants::OUTPUT_DIR;
use std::fs::{copy, create_dir, remove_file, write};
use std::path::Path;
use std::process::Command;
use std::str;
extern crate chrono;
use chrono::Local;

const HASHED: &str = "last";

fn kill(pid: &str, signal: &str) -> bool {
    let output = Command::new("kill")
        .arg(signal)
        .arg(pid)
        .status()
        .expect("kill command failed");
    return output.success();
}

fn signal(name: &str, signal: &str) -> bool {
    let output = Command::new("pidof")
        .arg(name)
        .output()
        .expect("pidof command failed");
    let s = match str::from_utf8(&output.stdout) {
        Ok(v) => v,
        Err(e) => {
            println!("signal failed: {}", name);
            println!("{}", e);
            return false;
        }
    };
    let parts: Vec<&str> = s.split_whitespace().collect();
    let mut valid = true;
    for p in parts {
        if !kill(p, &format!("-{}", signal)) {
            valid = false;
        }
    }
    return valid;
}

fn signal_all() -> bool {
    if !signal("hostapd", "HUP") {
        return false;
    }
    if !signal("radiucal", "2") {
        return false;
    }
    return true;
}

pub fn run_configuration(client: bool) -> bool {
    println!("updating networking configuration");
    let outdir = Path::new(OUTPUT_DIR);
    if !outdir.exists() {
        create_dir(outdir).expect("unable to make output directory");
    }
    let hash = outdir.join(HASHED);
    let prev_hash = outdir.join(HASHED.to_owned() + ".prev");
    if hash.exists() {
        copy(hash, prev_hash).expect("unable to maintain last hash");
    }
    let changed = outdir.join("changed");
    if changed.exists() {
        remove_file(changed).expect("unable to clear change cache");
    }
    // TODO: run netconf processing of inputs
    let server = !client;
    let date = Local::now().format(".radius.%Y-%m-%d").to_string();
    if server {
        println!("checking for daily operations");
        let daily = Path::new("/tmp/").join(date);
        if !daily.exists() {
            println!("running daily operations");
            if !signal_all() {
                println!("failed signaling daily");
            }
            match write(daily, "done") {
                Ok(_) => {}
                Err(e) => {
                    println!("unable to write daily indicator {}", e);
                }
            }
        }
    }

    return true;
}
