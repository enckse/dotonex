use md4::{Digest, Md4};
extern crate rand;
use rand::distributions::Alphanumeric;
use rand::Rng;
use std::fs::{File, OpenOptions};
use std::io;
use std::io::prelude::*;
use std::path::Path;

const CONFIG_DIR: &str = "config";

fn process<D: Digest + Default>(value: &str) -> String {
    let mut sh: D = Default::default();
    sh.input(value);
    let r = &sh.result();
    let mut buf = String::with_capacity(r.len());
    for byte in r {
        buf.push_str(&format!("{:02x}", byte));
    }
    return buf;
}

fn md4_hash(value: &str) -> String {
    return process::<Md4>(value);
}

fn generate_password(input_password: &str, out_password: &mut String) -> String {
    if input_password == "" {
        let pass = rand::thread_rng()
            .sample_iter(&Alphanumeric)
            .take(64)
            .collect::<String>();
        out_password.push_str(&pass);
    } else {
        out_password.push_str(input_password);
    }
    return md4_hash(&out_password);
}

pub fn new_user(user_name: &str, input_password: &str) -> Result<bool, io::Error> {
    let mut user = user_name;
    let mut input = String::new();
    if user_name == "" {
        println!("please provide user name:");
        match io::stdin().read_line(&mut input) {
            Ok(_) => {
                user = &input;
            }
            Err(error) => return Err(error),
        }
        user = user.trim();
        if user == "" {
            println!("empty username");
            return Ok(false);
        }
    }
    for c in user.chars() {
        if c >= 'a' && c <= 'z' {
            continue;
        }
        println!("invalid user name (a-z): {}", user);
        return Ok(false);
    }
    let mut out = String::new();
    let md4 = generate_password(input_password, &mut out);
    println!("username: {}\npassword: {}\nmd4 hash: {}", user, out, md4);
    let mut user_file = String::new();
    user_file.push_str("user_");
    user_file.push_str(user);
    user_file.push_str(".yaml");
    let user_path = Path::new(CONFIG_DIR).join(user_file);
    let mut buffer = File::create(user_path)?;
    buffer.write(b"user:\n")?;
    let pass_file = Path::new(CONFIG_DIR).join("passwords");
    let mut file = OpenOptions::new()
        .write(true)
        .append(true)
        .open(pass_file)?;
    file.write_fmt(format_args!("{},{}\n", user, md4))?;
    return Ok(true);
}
