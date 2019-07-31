use md4::{Digest, Md4};
extern crate rand;
use crate::constants::{random_string, CONFIG_DIR, PASSWORDS};
use crate::encrypt::{decrypt_file, encrypt_file};
use csv::{ReaderBuilder, Writer};
use encoding::all::UTF_16LE;
use encoding::{EncoderTrap, Encoding};
use std::fs::File;
use std::io;
use std::io::prelude::*;
use std::path::Path;

/// Process a password into a digest hash output
fn process<D: Digest + Default>(value: &str) -> String {
    let mut sh: D = Default::default();
    let utf16 = UTF_16LE
        .encode(value, EncoderTrap::Ignore)
        .expect("utf-16le failure");
    sh.input(utf16);
    let r = &sh.result();
    let mut buf = String::with_capacity(r.len());
    for byte in r {
        buf.push_str(&format!("{:02x}", byte));
    }
    return buf;
}

/// md4 hashing
fn md4_hash(value: &str) -> String {
    return process::<Md4>(value);
}

/// use or generate a password
fn generate_password(out_password: &mut String) -> String {
    let pass: String = random_string(64);
    out_password.push_str(&pass);
    let md4 = md4_hash(&out_password);
    println!("password: {}\nmd4 hash: {}", out_password, md4);
    return md4;
}

// read a username from stdin
fn read_username() -> Option<String> {
    let mut input = String::new();
    let mut user = String::new();;
    println!("please provide user name:");
    match io::stdin().read_line(&mut input) {
        Ok(_) => {
            user.push_str(&input);
        }
        Err(error) => {
            println!("unable to read stdin: {}", error);
            return None;
        }
    }
    user = user.trim().to_string();
    if user == "" {
        println!("empty username");
        return None;
    }
    return Some(user.to_string());
}

/// get a password as a hashed value
pub fn get_pass() -> bool {
    let mut out = String::new();
    generate_password(&mut out);
    return true;
}

/// create a new user
fn create_user() -> Result<bool, io::Error> {
    match read_username() {
        Some(user) => {
            for c in user.chars() {
                if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')  {
                    continue;
                }
                println!("invalid user name (a-z): {}", user);
                return Ok(false);
            }
            let mut out = String::new();
            println!("username: {}\n", user);
            let md4 = generate_password(&mut out);
            let mut user_file = String::new();
            user_file.push_str("user_");
            user_file.push_str(&user.to_string());
            user_file.push_str(".lua");
            let user_path = Path::new(CONFIG_DIR).join(user_file);
            let mut buffer = File::create(user_path)?;
            buffer.write(b"")?;
            return add_pass(user, md4);
        }
        None => {
            return Ok(false);
        }
    }
}

fn add_pass(user: String, md4: String) -> Result<bool, io::Error> {
    let pass_file = Path::new(CONFIG_DIR).join(PASSWORDS);
    let mut records: std::vec::Vec<std::vec::Vec<String>> = std::vec::Vec::new();
    if pass_file.exists() {
        let mut rdr = ReaderBuilder::new()
            .has_headers(false)
            .from_path(&pass_file)
            .expect("unable to read pass file");
        for result in rdr.records() {
            let record = result.expect("invalid csv entry");
            if record.len() != 2 {
                println!("invalid record {:?}", record);
                return Ok(false);
            }
            let r_user = record.get(0).expect("no user field found").to_owned();
            let r_pass = record.get(1).expect("no password field found").to_owned();
            if r_user == user {
                continue;
            }
            records.push(vec![r_user, r_pass]);
        }
    }
    records.push(vec![user, md4]);
    let mut wtr = Writer::from_path(&pass_file).expect("unable to write pass file");
    for r in records {
        wtr.write_record(r).expect("unable to write record to csv");
    }
    wtr.flush().expect("unable to save file");
    return Ok(true);
}

pub fn new_user(pass: &str) -> bool {
    if !decrypt_file(pass) {
        return false;
    }
    let status = create_user();
    match status {
        Ok(n) => {
            if n {
                return encrypt_file(pass);
            } else {
                return n;
            }
        }
        Err(error) => {
            println!("{}", error);
            return false;
        }
    }
}

pub fn passwd(pass: &str) -> bool {
    let mut valid = false;
    match read_username() {
        Some(u) => {
            let mut out = String::new();
            let md4 = generate_password(&mut out);
            if decrypt_file(&pass) {
                match add_pass(u, md4) {
                    Ok(ok) => {
                        if ok {
                            if encrypt_file(&pass) {
                                valid = true;
                            }
                        } else {
                            println!("unable to set password");
                        }
                    }
                    Err(e) => {
                        println!("error adding password {}", e);
                    }
                }
            }
        }
        None => {
            println!("invalid username");
        }
    }
    valid
}
