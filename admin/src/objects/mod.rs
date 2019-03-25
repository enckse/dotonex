extern crate yaml_rust;
use std::collections::{HashMap, HashSet};
use std::fs::File;
use std::io::prelude::*;
use std::path::PathBuf;
use yaml_rust::{Yaml, YamlLoader};

pub struct VLAN {
    pub name: String,
    pub number: i64,
    initiate: Vec<String>,
    net: String,
    owner: String,
    description: String,
    group: String,
    route: String,
}

pub struct Object {
    make: String,
    name: String,
    model: String,
    revision: String,
}

pub struct Device {
    make: String,
    name: String,
    model: String,
    revision: String,
    serial: String,
    macs: Vec<String>,
}

pub struct Assignment {
    vlan: String,
    mode: String,
    nodes: Vec<String>,
}

pub struct User {
    default: String,
    disabled: bool,
    devices: Vec<Device>,
    assignments: Vec<Assignment>,
}

impl Object {
    fn copy_to(&self, device: &mut Device) {
        device.make = self.make.to_owned();
        device.model = self.model.to_owned();
        device.revision = self.revision.to_owned();
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
    println!("reading: {}", file);
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

pub fn load_vlans(paths: Vec<PathBuf>) -> Result<HashMap<String, VLAN>, String> {
    let mut vlans: HashMap<String, VLAN> = HashMap::new();
    let mut vlan_nums: HashSet<i64> = HashSet::new();
    for p in paths {
        match p.file_name() {
            Some(n) => {
                println!("{}", n.to_string_lossy());
                if n.to_string_lossy().starts_with("vlan_") {
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
