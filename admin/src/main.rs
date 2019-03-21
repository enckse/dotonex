mod useradd;
use crate::useradd::generate_password;

fn main() {
    let mut pass = String::new();
    let md4 = generate_password("abc", &mut pass);
    println!("password:{}\nmd4:{}", pass, md4);
    pass = String::new();
    let md42 = generate_password("", &mut pass);
    println!("password:{}\nmd4:{}", pass, md42);
}
