mod useradd;
use crate::useradd::new_user;

fn main() {
    let valid = new_user("", "");
    match valid {
        Ok(n) => {
            if !n {
                std::process::exit(1);
            }
        }
        Err(error) => {
            println!("{}", error);
            std::process::exit(2);
        }
    }
}
