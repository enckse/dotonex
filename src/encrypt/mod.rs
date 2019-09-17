use crate::constants::{random_string, CONFIG_DIR, PASSWORDS};
use aes_ctr::stream_cipher::generic_array::GenericArray;
use aes_ctr::stream_cipher::{NewStreamCipher, SyncStreamCipher, SyncStreamCipherSeek};
use aes_ctr::Aes256Ctr;
use std::fs;
use std::path::Path;

const NONCE_SIZE: usize = 16;

fn get_pass_file() -> String {
    let mut file = String::new();
    file.push_str(PASSWORDS);
    file.push_str(".keys");
    file
}

fn encrypt_decrypt(pass: &str, decrypt: bool) -> bool {
    if pass.is_empty() {
        println!("no password given");
        return false;
    }
    let in_file = if decrypt {
        get_pass_file()
    } else {
        PASSWORDS.to_string()
    };
    let out_file = if decrypt {
        PASSWORDS.to_string()
    } else {
        get_pass_file()
    };
    let ifile = Path::new(CONFIG_DIR).join(in_file);
    let ofile = Path::new(CONFIG_DIR).join(out_file);
    let key = GenericArray::from_slice(pass.as_bytes());
    match fs::read(ifile) {
        Ok(data) => {
            let mut use_data: std::vec::Vec<u8>;
            let nonce_data: std::vec::Vec<u8>;
            if decrypt {
                nonce_data = data[0..NONCE_SIZE].to_vec();
                use_data = data[NONCE_SIZE..].to_vec();
            } else {
                use_data = data.to_vec();
                nonce_data = random_string(NONCE_SIZE).as_bytes().to_vec();
            }
            let nonce = GenericArray::from_slice(&nonce_data);
            let mut cipher = Aes256Ctr::new(&key, &nonce);
            if decrypt {
                cipher.seek(0);
            }
            cipher.apply_keystream(&mut use_data);
            let mut idx = 0;
            if !decrypt {
                for i in &nonce_data {
                    use_data.insert(idx, *i);
                    idx += 1;
                }
            }
            match fs::write(ofile, use_data) {
                Ok(_) => true,
                Err(e) => {
                    println!("unable to write file {}", e);
                    false
                }
            }
        }
        Err(e) => {
            println!("unable to read file {}", e);
            false
        }
    }
}

pub fn decrypt_file(pass: &str) -> bool {
    encrypt_decrypt(pass, true)
}

pub fn encrypt_file(pass: &str) -> bool {
    encrypt_decrypt(pass, false)
}
