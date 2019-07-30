use md4::{Digest, Md4};
extern crate rand;
use crate::constants::{random_string, CONFIG_DIR, PASSWORDS};
use encoding::all::UTF_16LE;
use encoding::{EncoderTrap, Encoding};
use std::fs::{File, OpenOptions};
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
pub fn generate_password(input_password: &str, out_password: &mut String) -> String {
    if input_password == "" {
        let pass: String = random_string(64);
        out_password.push_str(&pass);
    } else {
        out_password.push_str(input_password);
    }
    return md4_hash(&out_password);
}

// read a username from stdin
pub fn read_username() -> Option<String> {
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
pub fn get_pass(pass: &str) -> bool {
    let mut out = String::new();
    let md4 = generate_password(pass, &mut out);
    println!("password: {}\nmd4 hash: {}", out, md4);
    return true;
}

/// create a new user
fn create_user(user_name: &str, input_password: &str) -> Result<bool, io::Error> {
    let mut user = String::new();
    user.push_str(user_name);
    if user_name == "" {
        match read_username() {
            Some(u) => {
                user = u;
            }
            None => {
                return Ok(false);
            }
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
    user_file.push_str(&user.to_string());
    user_file.push_str(".yaml");
    let user_path = Path::new(CONFIG_DIR).join(user_file);
    let mut buffer = File::create(user_path)?;
    buffer.write(b"")?;
    return add_pass(user, md4);
}

pub fn add_pass(user: String, md4: String) -> Result<bool, io::Error> {
    let pass_file = Path::new(CONFIG_DIR).join(PASSWORDS);
    let mut file = OpenOptions::new()
        .write(true)
        .append(true)
        .open(pass_file)?;
    file.write_fmt(format_args!("{},{}\n", user, md4))?;
    return Ok(true);
}

pub fn new_user(user_name: &str, pass: &str) -> bool {
    let status = create_user(user_name, pass);
    match status {
        Ok(n) => {
            return n;
        }
        Err(error) => {
            println!("{}", error);
            return false;
        }
    }
}
