use md4::{Md4, Digest};

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


fn main() {
    println!("{}", process::<Md4>("abc"));
}
