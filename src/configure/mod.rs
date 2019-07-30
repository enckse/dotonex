use crate::constants::{CONFIG_DIR, EAP_USERS, MANIFEST, OUTPUT_DIR};
use std::fs::{copy, create_dir, create_dir_all, remove_file, write};
use std::path::{Path, PathBuf};
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

fn update(outdir: PathBuf) -> bool {
    let manifest = outdir.join(MANIFEST);
    if !manifest.exists() {
        println!("missing manifest file");
        return false;
    }
    let var_lib = Path::new("/var/lib/radiucal/");
    let var_home = var_lib.join("users");
    if !var_home.exists() {
        create_dir_all(&var_home).expect("unable to make live configs");
    }
    let contents = fs::read_to_string(manifest).expect("unable to read manifest");
    let base_users: std::vec::Vec<&str> = contents.split("\n").collect();
    let mut new_users: std::vec::Vec<String> = std::vec::Vec::new();
    for b in base_users {
        if b == "" {
            continue;
        }
        new_users.push(var_home.join(b).to_string_lossy().into_owned());
    }
    let cur_users = get_file_list(&var_home.to_string_lossy().into_owned());
    for u in cur_users {
        match new_users.iter().position(|r| r == &u) {
            Some(_) => {}
            None => {
                println!("dropping file {}", u);
                remove_file(u).expect("unable to remove file");
            }
        }
    }
    for u in new_users {
        if u == "" {
            continue;
        }
        let user_file = Path::new(&u);
        if !user_file.exists() {
            fs::write(user_file, "user").expect("unable to write file");
        }
    }
    let eap_bin = outdir.join(EAP_USERS);
    let eap_var = var_lib.join(EAP_USERS);
    if !eap_bin.exists() {
        println!("eap_users file is missing?");
        return false;
    }
    copy(eap_bin, eap_var).expect("unable to copy eap_users file");
    return signal_all();
}

fn get_file_list(dir: &str) -> std::vec::Vec<String> {
    let mut file_list: std::vec::Vec<String> = std::vec::Vec::new();
    let files = fs::read_dir(dir).expect("unable to read directory");
    for f in files {
        let entry = f.expect("unable to read dir");
        file_list.push(entry.path().to_string_lossy().into_owned());
    }
    return file_list;
}

pub fn netconf() -> bool {
    let output = Command::new("radiucal-lua-bridge")
        .status()
        .expect("lua-bridge command failed");
    return output.success();
}

pub fn all(server: bool) -> bool {
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
    if !netconf() {
        return false;
    }
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
    let mut file_list = get_file_list(CONFIG_DIR);
    file_list.sort();
    let output = Command::new("sha256sum")
        .args(file_list)
        .output()
        .expect("sha256sum command failed");
    fs::write(&hash, output.stdout).expect("unable to store hashes");
    if hash.exists() && prev_hash.exists() {
        let output = Command::new("diff")
            .arg("-u")
            .arg(prev_hash)
            .arg(hash)
            .status()
            .expect("diff command failed");
        diffed = !output.success();
    }
    if diffed {
        println!("configuration updated");
        if server {
            return update(outdir.to_path_buf());
        }
    }
    return true;
}
