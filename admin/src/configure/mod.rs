use crate::constants::{CONFIG_DIR, OUTPUT_DIR};
use std::fs::{copy, create_dir, write};
use std::path::Path;
use std::process::Command;
use std::str;
extern crate chrono;
use chrono::Local;
use std::fs;

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
        copy(&hash, &prev_hash).expect("unable to maintain last hash");
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

    let mut diffed = true;
    let files = fs::read_dir(CONFIG_DIR).expect("unable to read config directory");
    let mut file_list: std::vec::Vec<String> = std::vec::Vec::new();
    for f in files {
        let entry = f.expect("unable to read dir");
        file_list.push(entry.path().to_string_lossy().into_owned());
    }
    file_list.sort();
    let output = Command::new("sha256sum")
        .args(file_list)
        .output()
        .expect("sha256sum command failed");
    fs::write(&hash, output.stdout).expect("unable to store hashes");
    if hash.exists() && prev_hash.exists() {
        let output = Command::new("diff")
            .arg("-u")
            .arg(hash)
            .arg(prev_hash)
            .status()
            .expect("diff command failed");
        diffed = !output.success();
    }
    if diffed {
        println!("configuration updated");
        if server {
            // TODO: run update commands
        }
    }
    return true;
}
