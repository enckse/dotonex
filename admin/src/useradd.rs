use md4::{Md4, Digest};
extern crate rand;
use rand::Rng;
use rand::distributions::Alphanumeric;

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

pub fn generate_password(input_password: &str, out_password: &mut String) -> String {
    if input_password == "" {
        let pass = rand::thread_rng().sample_iter(&Alphanumeric).take(64).collect::<String>();
        out_password.push_str(&pass);
    } else {
        out_password.push_str(input_password);
    }
    return md4_hash(&out_password);
}
