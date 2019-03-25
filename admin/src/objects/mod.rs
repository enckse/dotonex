extern crate yaml_rust;
use crate::constants::OUTPUT_DIR;
use std::collections::{HashMap, HashSet};
use std::fs::File;
use std::io::prelude::*;
use std::path::{Path, PathBuf};
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

impl VLAN {
    fn to_markdown(&self) -> String {
        return format!(
            "| {} | {} | {} | {} | {} | {} |\n",
            self.group, self.name, self.net, self.number, self.owner, self.description
        );
    }
    fn to_diagram(&self) -> String {
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
    let out = Path::new(OUTPUT_DIR);
    let mut dot =
        File::create(out.join("segment-diagram.dot")).expect("unable to create dot diagram");
    let mut md = File::create(out.join("segments.md")).expect("unable to create segments markdown");
    dot.write(
        b"digraph g {
    size=\"6,6\";
    node [color=lightblue2, style=filled];
",
    )
    .expect("dot header failed");
    md.write(
        b"| cell | segment | lan | vlan | owner | description |
| --- | --- | --- | --- | --- | --- |
",
    )
    .expect("md header failed");
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
                    dot.write(v.to_diagram().as_bytes())
                        .expect("could not write to dot file");
                    md.write(v.to_markdown().as_bytes())
                        .expect("could not write to md file");
                    vlans.insert(v.name.to_owned(), v);
                }
            }
            None => continue,
        }
    }
    dot.write(b"}\n").expect("unable to close dot file");
    Ok(vlans)
}

pub fn load_objects(file: String) -> Result<HashMap<String, Object>, String> {
    let doc = load_yaml(file);
    let mut objs: HashMap<String, Object> = HashMap::new();
    match doc["objects"].as_hash() {
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
