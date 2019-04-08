extern crate yaml_rust;
use crate::constants::{CONFIG_DIR, IS_YAML};
use std::collections::{HashMap, HashSet};
use std::fs::File;
use std::io::prelude::*;
use std::path::PathBuf;
use yaml_rust::{Yaml, YamlLoader};

const USER_INDICATOR: &str = "user_";
const IS_DISABLE: &str = "user is disabled";

pub struct VLAN {
    pub name: String,
    pub number: i64,
    pub initiate: Vec<String>,
    net: String,
    owner: String,
    description: String,
    group: String,
    route: String,
}

pub struct Password {
    pub pass: String,
}

pub struct Object {
    pub make: String,
    pub name: String,
    pub model: String,
    pub revision: String,
}

pub struct Assignment {
    pub vlan: String,
    pub mode: String,
}

pub struct Device {
    pub serial: String,
    pub name: String,
    pub base: String,
    pub macs: HashMap<String, Assignment>,
}

pub struct User {
    pub name: String,
    pub default_vlan: String,
    pub devices: Vec<Device>,
}

fn check_lowercase(value: &String) -> bool {
    let mut valid = true;
    if value.is_empty() {
        return false;
    }
    for c in value.as_str().chars() {
        if c >= 'a' && c <= 'z' {
            continue;
        }
        valid = false;
    }
    valid
}

impl Device {
    fn check(&self) -> Option<&str> {
        if self.name == "" {
            return Some("device name is empty");
        }
        if self.base == "" {
            return Some("device base is empty");
        }
        if self.serial == "" {
            return Some("device serial is empty");
        }
        let mut has_macs = false;
        let mut invalid_count = 0;
        for m in self.macs.keys() {
            has_macs = true;
            let mut is_valid = true;
            if m.len() == 12 {
                for c in m.as_str().chars() {
                    if (c >= 'a' && c <= 'f') || (c >= '0' && c <= '9') {
                        continue;
                    }
                    is_valid = false;
                }
                if is_valid {
                    let assign = &self.macs[m];
                    if !check_lowercase(&assign.mode) {
                        return Some("invalid mode on mac");
                    }
                    if !check_lowercase(&assign.vlan) {
                        return Some("invalid vlan on mac");
                    }
                }
            } else {
                is_valid = false;
            }
            if !is_valid {
                println!("{} is an invalid mac...", m);
                invalid_count += 1
            }
        }
        if invalid_count > 0 {
            return Some("^^^^^ invalid macs detected ^^^^^");
        }
        if !has_macs {
            return Some("device has no macs");
        }
        None
    }
}

impl User {
    fn check(&self) -> Option<&str> {
        if !check_lowercase(&self.name) {
            return Some("invalid name (a-z)");
        }
        if !check_lowercase(&self.default_vlan) {
            return Some("invalid vlan (a-z)");
        }
        let mut has_dev = false;
        for d in &self.devices {
            has_dev = true;
            let d_check = d.check();
            match d_check {
                Some(_) => {
                    return d_check;
                }
                None => {}
            }
        }
        if !has_dev {
            return Some("no devices");
        }
        None
    }
}

impl VLAN {
    pub fn to_markdown(&self) -> String {
        return format!(
            "| {} | {} | {} | {} | {} | {} |\n",
            self.group, self.name, self.net, self.number, self.owner, self.description
        );
    }
    pub fn to_diagram(&self) -> String {
        let mut result: String = String::new();
        result.push_str(&format!("    \"{}\" [shape=\"record\"]\n", self.name));
        if self.route != "none" {
            result.push_str(&format!(
                "    \"{}\" -> \"{}\" [color=red]\n",
                self.name, self.route
            ));
        }
        if self.initiate.len() > 0 {
            for i in &self.initiate {
                result.push_str(&format!("    \"{}\" -> \"{}\"\n", self.name, i));
            }
        }
        result
    }
}

fn optional_field(doc: &Yaml, key: &str) -> String {
    match doc[key].as_str() {
        Some(v) => v.to_string(),
        None => String::new(),
    }
}

fn load_yaml(file: String) -> Yaml {
    let mut f = File::open(file).expect("unable to load file");
    let mut buffer = String::new();
    f.read_to_string(&mut buffer).expect("unable to read file");
    let docs = YamlLoader::load_from_str(&buffer).expect("unable to parse yaml");
    return docs[0].clone();
}

fn load_vlan(file: String) -> Result<VLAN, String> {
    let doc = load_yaml(file);
    let mut initiate: Vec<String> = Vec::new();
    match doc["initiate"].as_vec() {
        Some(vector) => {
            for a in vector {
                initiate.push(a.as_str().expect("invalid initiate").to_string());
            }
        }
        None => {}
    }
    let vlan = VLAN {
        name: doc["name"].as_str().expect("invalid vlan name").to_string(),
        description: optional_field(&doc, "description"),
        net: optional_field(&doc, "net"),
        owner: optional_field(&doc, "owner"),
        route: optional_field(&doc, "route"),
        group: optional_field(&doc, "group"),
        number: doc["number"].as_i64().expect("invalid number"),
        initiate: initiate,
    };
    if vlan.number < 0 || vlan.number > 4096 || vlan.name == "" {
        return Err(format!("invalid vlan definition"));
    }
    return Ok(vlan);
}

pub fn load_vlans(paths: &Vec<PathBuf>) -> Result<HashMap<String, VLAN>, String> {
    let mut vlans: HashMap<String, VLAN> = HashMap::new();
    let mut vlan_nums: HashSet<i64> = HashSet::new();
    for p in paths {
        match p.file_name() {
            Some(n) => {
                if n.to_string_lossy().starts_with("vlan_") {
                    println!("vlan: {}", n.to_string_lossy());
                    let v = load_vlan(p.to_string_lossy().to_string())?;
                    if vlans.contains_key(&v.name) {
                        return Err(format!("vlan redefined {}", v.name));
                    }
                    if vlan_nums.contains(&v.number) {
                        return Err(format!("vlan redefined {}", v.number));
                    }
                    vlan_nums.insert(v.number.to_owned());
                    vlans.insert(v.name.to_owned(), v);
                }
            }
            None => continue,
        }
    }
    Ok(vlans)
}

pub fn load_objects(file: String) -> Result<HashMap<String, Object>, String> {
    let doc = load_yaml(file);
    let mut objs: HashMap<String, Object> = HashMap::new();
    match doc.as_hash() {
        Some(hash) => {
            for o in hash {
                let obj = Object {
                    name: o.0.as_str().expect("missing name field").to_string(),
                    make: o.1["make"]
                        .as_str()
                        .expect("missing make field")
                        .to_string(),
                    model: o.1["model"]
                        .as_str()
                        .expect("missing model field")
                        .to_string(),
                    revision: optional_field(&doc, "revision"),
                };
                objs.insert(obj.name.to_owned(), obj);
            }
        }
        None => {}
    }
    Ok(objs)
}

fn load_user(file: String) -> Result<User, String> {
    let name = file
        .replace(USER_INDICATOR, "")
        .replace(IS_YAML, "")
        .replace(CONFIG_DIR, "")
        .replace("/", "");
    let doc = load_yaml(file);
    let user_def = &doc["user"];
    let default_vlan = user_def["vlan"]
        .as_str()
        .expect("default vlan required")
        .to_string();
    let disabled = match user_def["disabled"].as_bool() {
        Some(b) => b,
        None => false,
    };
    if disabled {
        return Err(IS_DISABLE.to_string());
    }
    let mut devices: Vec<Device> = Vec::new();
    let objects = &doc["objects"].as_hash().expect("no object definitions");
    for o in objects.keys() {
        let key = o.as_str().expect("invalid object id");
        let obj = objects[o]
            .as_hash()
            .expect("object definition is not a hash/dictionary");
        let mut macs: HashMap<String, Assignment> = HashMap::new();
        let mut base = "";
        let mut serial = "";
        for obj_key in obj.keys() {
            let raw_key = obj_key
                .as_str()
                .expect("invalid object definition key is not string");
            let mut add_vlans: Vec<String> = Vec::new();
            let mut add_macs: HashMap<String, String> = HashMap::new();
            match raw_key {
                "base" => {
                    base = obj[obj_key].as_str().expect("invalid object base");
                }
                "serial" => {
                    serial = obj[obj_key].as_str().expect("invalid object serial");
                }
                "vlans" => {
                    for v in obj[obj_key]
                        .as_vec()
                        .expect("vlan setting must be an array")
                    {
                        add_vlans.push(v.as_str().expect("vlan must be a string").to_string());
                    }
                }
                "macs" => {
                    let mac_objs = obj[obj_key].as_hash().expect("invalid mac set");
                    for mac_key in mac_objs.keys() {
                        let mac_value = mac_key.as_str().expect("mac is invalid (not string)");
                        if macs.contains_key(mac_value) {
                            return Err(format!("device mac not unique: {}", mac_value));
                        }
                        let mut mode = String::new();
                        let mac_obj = &mac_objs[mac_key];
                        match mac_obj.as_str() {
                            Some(v) => {
                                mode = v.to_string();
                            }
                            None => {
                                return Err(format!("invalid mac object definition {}", mac_value));
                            }
                        }
                        if add_macs.contains_key(mac_value) {
                            return Err(format!("mac {} is defined multiple times", mac_value));
                        }
                        add_macs.insert(mac_value.to_owned(), mode);
                    }
                }
                _ => {
                    return Err(format!("unknown key: {}", raw_key));
                }
            }
            if add_vlans.len() == 0 {
                add_vlans.push(default_vlan.to_owned());
            }
            for v in add_vlans {
                for m in add_macs.keys() {
                    let mode = add_macs.get(m).expect("mode missing, internal error");
                    let assigned = Assignment {
                        mode: mode.to_string(),
                        vlan: v.to_owned(),
                    };
                    macs.insert(m.to_string(), assigned);
                }
            }
        }
        let device = Device {
            macs: macs,
            name: key.to_owned(),
            serial: serial.to_string(),
            base: base.to_string(),
        };
        devices.push(device);
    }
    let user = User {
        name: name,
        default_vlan: default_vlan,
        devices: devices,
    };
    match user.check() {
        Some(err) => Err(err.to_string()),
        None => Ok(user),
    }
}

pub fn load_passwords(file: String) -> Result<HashMap<String, Password>, String> {
    let doc = load_yaml(file);
    let mut objs: HashMap<String, Password> = HashMap::new();
    match doc.as_hash() {
        Some(hash) => {
            for o in hash {
                let name = o.0.as_str().expect("invalid user name").to_string();
                let pass = o.1.as_str().expect("invalid user password").to_string();
                let obj = Password { pass: pass };
                if objs.contains_key(&name) {
                    return Err(format!("{} already has password", name));
                }
                objs.insert(name.to_owned(), obj);
            }
        }
        None => {}
    }
    Ok(objs)
}

pub fn load_users(paths: &Vec<PathBuf>) -> Result<HashMap<String, User>, String> {
    let mut users: HashMap<String, User> = HashMap::new();
    for p in paths {
        match p.file_name() {
            Some(n) => {
                if n.to_string_lossy().starts_with(USER_INDICATOR) {
                    println!("user: {}", n.to_string_lossy());
                    let u = match load_user(p.to_string_lossy().to_string()) {
                        Ok(def) => Ok(def),
                        Err(e) => {
                            if e == IS_DISABLE {
                                println!("^^^ disabled ^^^");
                                continue;
                            } else {
                                Err(e)
                            }
                        }
                    }?;
                    if users.contains_key(&u.name) {
                        return Err(format!("user redefined {}", u.name));
                    }
                    users.insert(u.name.to_owned(), u);
                }
            }
            None => continue,
        }
    }
    Ok(users)
}
