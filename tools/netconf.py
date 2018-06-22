#!/usr/bin/python
"""composes the config from user definitions."""
import argparse
import os
import users
import importlib
import csv
from datetime import datetime
import re

# file indicators
IND_DELIM = "_"
USER_INDICATOR = "user" + IND_DELIM
VLAN_INDICATOR = "vlan" + IND_DELIM
AUTH_PHASE_ONE = "PEAP"
AUTH_PHASE_TWO = "MSCHAPV2"


def is_valid_mac(possible_mac):
    """check if an object is a mac."""
    valid = False
    if len(possible_mac) == 12:
        valid = True
        for c in possible_mac:
            if c not in ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
                         'a', 'b', 'c', 'd', 'e', 'f']:
                valid = False
                break
    return valid


def is_mac(value, category=None):
    """validate if something appears to be a mac."""
    valid = is_valid_mac(value)
    if not valid:
        cat = ''
        if category is not None:
            cat = " (" + category + ")"
        print('invalid mac detected{} {}'.format(cat, value))
    return valid


class VLAN(object):
    """VLAN definition."""

    def __init__(self, name, number):
        """init the instance."""
        self.name = name
        self.num = number
        self.initiate = None
        self.route = None
        self.net = None
        self.owner = None
        self.desc = None
        self.group = None

    def check(self, ):
        """Check the definition."""
        if self.name is None or len(self.name) == 0 \
           or not isinstance(self.num, int):
            return False
        return True


class Assignment(object):
    """assignment object."""

    def __init__(self):
        """Init the instance."""
        self.macs = []
        self.password = ""
        self.vlan = None
        self.expires = None
        self.disabled = False
        self.inherits = None
        self.owns = []
        self._bypass = None
        self.management = None
        self.mab_only = False

    def _compare_date(self, value, regex, today):
        """compare date."""
        matches = regex.findall(value)
        for match in matches:
            as_date = datetime.strptime(match, '%Y-%m-%d')
            return as_date < today
        return None

    def report(self, cause):
        """report an issue."""
        print(cause)
        return False

    def copy(self, other):
        """copy/inherit from another entity."""
        self.password = other.password
        self.macs = set(self.macs + other.macs)

    def mab(self, macs, vlan=None):
        """Set a MAC as MAB."""
        v = vlan
        if vlan is None:
            v = self.vlan
        if v is None:
            raise Exception("mab before vlan assigned")
        if self._bypass is None:
            self._bypass = {}
        m = [macs]
        if isinstance(macs, list):
            m = macs
        for mac in m:
            self._bypass[mac] = v

    def bypassed(self):
        """Get MAB bypassed MACs."""
        if self._bypass is None:
            return []
        return list(self._bypass.keys())

    def bypass_vlan(self, mac):
        """Get a MAB bypassed VLAN."""
        return self._bypass[mac]

    def _check_macs(self, against, previous=[]):
        """Check macs."""
        if against is not None and len(against) > 0:
            already_set = self.macs + previous
            if self._bypass is not None:
                already_set = already_set + list(self._bypass.keys())
            if previous is not None:
                already_set = already_set + previous
            for mac in against:
                if not is_mac(mac):
                    return mac
                if mac in already_set:
                    return mac
        return None

    def check(self):
        """check the assignment definition."""
        if self.inherits:
            self.copy(self.inherits)
        today = datetime.now()
        today = datetime(today.year, today.month, today.day)
        regex = re.compile(r'\d{4}[-/]\d{2}[-/]\d{2}')
        if self.expires is not None:
            res = self._compare_date(self.expires, regex, today)
            if res is not None:
                self.disabled = res
            else:
                return self.report("invalid expiration")
        if self.vlan is None or len(self.vlan) == 0:
            return self.report("no vlan assigned")
        has_mac = False
        knowns = []
        if self._bypass is not None:
            knowns = self._bypass.keys()
        for mac_group in [self.macs, knowns]:
            if mac_group is not None and len(mac_group) > 0:
                has_mac = True
        if not self.disabled and not has_mac:
            return self.report("no macs listed")
        for mac in self.macs:
            if not is_mac(mac):
                return False
        if self.password is None or len(self.password) == 0:
            return self.report("no or short password")
        if len(knowns) > 0:
            for mac in knowns:
                if not is_mac(mac, category='bypass'):
                    return False
        for c in [self._check_macs(self.owns)]:
            if c is not None:
                return self.report("invalid mac (known): {}".format(c))
        if len(self.macs) != len(set(self.macs)):
            return self.report("macs not unique")
        if self.management is not None:
            if isinstance(self.management, list):
                return self.report("management can only be one mac")
            if not is_valid_mac(self.management):
                return self.report("invalid management mac")
        return True


class ConfigMeta(object):
    """configuration meta information."""

    def __init__(self):
        """init the instance."""
        self.passwords = []
        self.macs = []
        self.vlans = []
        self.all_vlans = []
        self.user_name = []
        self.vlan_users = []
        self.vlan_initiate = []
        self.extras = []

    def password(self, password):
        """password group validation(s)."""
        if password in self.passwords:
            print("password duplicated")
            exit(-1)
        self.passwords.append(password)

    def extra(self, macs):
        """Limited macs."""
        for mac in macs:
            if mac in self.extras:
                print("mac already known as extra: " + mac)
                exit(-1)
            self.extras.append(mac)

    def user_macs(self, macs):
        """user+mac combos."""
        self.macs = self.macs + macs
        self.macs = list(set(self.macs))

    def verify(self):
        """verify meta data."""
        for mac in self.macs:
            if mac in self.extras:
                print("mac is flagged extra: " + mac)
                exit(-1)
        for mac in self.extras:
            if mac in self.macs:
                print("mac is user assigned: " + mac)
                exit(-1)
        used_vlans = set(self.vlans + self.vlan_initiate)
        if len(used_vlans) != len(set(self.all_vlans)):
            print("unused vlans detected")
            exit(-1)
        for ref in used_vlans:
            if ref not in self.all_vlans:
                print("reference to unknown vlan: " + ref)
                exit(-1)

    def vlan_user(self, vlan, user):
        """indicate a vlan was used."""
        self.vlans.append(vlan)
        self.vlan_users.append(vlan + "." + user)
        self.user_name.append(user)

    def vlan_to_vlan(self, vlan_to):
        """VLAN to VLAN mappings."""
        self.vlan_initiate.append(vlan_to)


def _get_mod(name):
    """import the module dynamically."""
    return importlib.import_module("users." + name)


def _load_objs(name, typed):
    mod = _get_mod(name)
    for key in dir(mod):
        obj = getattr(mod, key)
        if type(obj).__name__ != typed.__name__:
            continue
        yield obj


def _get_by_indicator(indicator):
    """get by a file type indicator."""
    y = []
    for p in os.listdir("users"):
        if p.startswith(indicator):
            y.append(p.replace(".py", ""))
    return y


def _common_call(common, method, entity):
    """make a common mod call."""
    obj = entity
    if common is not None and method in dir(common):
        call = getattr(common, method)
        if call is not None:
            obj = call(obj)
    return obj


def check_object(obj):
    """Check an object."""
    return obj.check()


def _process(output):
    """process the composition of users."""
    common_mod = None
    try:
        common_mod = _get_mod("common")
        print("loaded common definitions...")
    except Exception as e:
        print("defaults only...")
    vlans = None
    meta = ConfigMeta()
    for v_name in _get_by_indicator(VLAN_INDICATOR):
        print("loading vlan..." + v_name)
        for obj in _load_objs(v_name, VLAN):
            if vlans is None:
                vlans = {}
            if not check_object(obj):
                exit(-1)
            num_str = str(obj.num)
            for vk in vlans.keys():
                if num_str == vlans[vk]:
                    print("vlan number defined multiple times...")
                    exit(-1)
            vlans[obj.name] = num_str
            if obj.initiate is not None and len(obj.initiate) > 0:
                for init_to in obj.initiate:
                    meta.vlan_to_vlan(init_to)
    if vlans is None:
        raise Exception("missing required config settings...")
    meta.all_vlans = vlans.keys()
    store = Store()
    admins = {}
    for f_name in _get_by_indicator(USER_INDICATOR):
        print("composing..." + f_name)
        for obj in _load_objs(f_name, Assignment):
            obj = _common_call(common_mod, 'ready', obj)
            key = f_name.replace(USER_INDICATOR, "")
            if not key.isalnum():
                print("does not meet naming requirements...")
                exit(-1)
            vlan = obj.vlan
            if vlan not in vlans:
                raise Exception("no vlan defined for " + key)
            store.add_vlan(vlan, vlans[vlan])
            meta.vlan_user(vlan, key)
            fqdn = vlan + "." + key
            if not check_object(obj):
                print("did not pass check...")
                exit(-1)
            if obj.disabled:
                print("account is disabled")
                continue
            macs = sorted(obj.macs)
            password = obj.password
            bypassed = sorted(obj.bypassed())
            owned = sorted(obj.owns)
            # meta checks
            meta.user_macs(macs)
            if not obj.inherits:
                meta.password(password)
            meta.extra(bypassed)
            meta.extra(owned)
            store.add_user(fqdn, macs, password)
            if obj.mab_only:
                store.set_mab(fqdn)
            if len(bypassed) > 0:
                for m in bypassed:
                    store.add_mab(m, obj.bypass_vlan(m))
            user_all = []
            for l in [obj.macs, obj.owns, bypassed]:
                user_all += list(l)
            user_set = sorted(set(user_all))
            store.add_audit(fqdn, sorted(set(user_all)))
            if obj.management is not None:
                admin_mac = [obj.management]
                # we need to replace this into the store
                admins[fqdn] = [admin_mac, password]
    meta.verify()
    if len(admins) > 0:
        v_names = store.get_vlan_names()
        for named in admins:
            admin = admins[named]
            a = named.split(".")[1]
            m = admin[0]
            for v in v_names:
                fqdn = "{}.{}".format(v, a)
                # we already have a specified account for the admin in the vlan
                if fqdn == named:
                    continue
                store.add_user(fqdn, m, admin[1])
                store.add_audit(fqdn, m)

    # audit outputs
    with open(output + "audit.csv", 'w') as f:
        csv_writer = csv.writer(f, lineterminator=os.linesep)
        for a in sorted(store.get_tag(store.audit)):
            p = a[0].split(".")
            for m in a[1]:
                csv_writer.writerow([p[1], p[0], m])
    # eap_users and preauth
    manifest = []
    with open(output + "eap_users", 'w') as f:
        for u in store.get_eap_user():
            f.write('"{}" {}\n\n'.format(u[0], AUTH_PHASE_ONE))
            f.write('"{}" {} hash:{} [2]\n'.format(u[0], AUTH_PHASE_TWO, u[1]))
            write_vlan(f, u[2])
        for u in store.get_eap_mab():
            up = u[0].upper()
            f.write('"{}" MD5 "{}"\n'.format(up, up))
            write_vlan(f, u[1])
            manifest.append((u[0], u[0]))
    for u in store.get_tag(store.umac):
        manifest.append((u[0], u[1]))
    with open(output + "manifest", 'w') as f:
        for m in sorted(manifest):
            f.write("{}.{}\n".format(m[0], m[1]).lower())


def write_vlan(f, vlan_id):
    """Write vlan assignment for login."""
    f.write('radius_accept_attr=64:d:13\n')
    f.write('radius_accept_attr=65:d:6\n')
    f.write('radius_accept_attr=81:s:{}\n\n'.format(vlan_id))


class Store(object):
    """Storage object."""

    def __init__(self):
        """Init the instance."""
        self._data = []
        self.umac = "UMAC"
        self.pwd = "PWD"
        self.mac = "MAC"
        self.audit = "AUDIT"
        self._users = []
        self._mab = []
        self._macs = []
        self._vlans = {}

    def set_mab(self, username):
        """Set a user as MAB-only, no login set."""
        self._mab.append(username)

    def get_tag(self, tag):
        """Get tagged items."""
        for item in self._data:
            if item[0] == tag:
                yield item[1:]

    def add_vlan(self, vlan_name, vlan_id):
        """Add a vlan item."""
        self._vlans[vlan_name] = vlan_id

    def _add(self, tag, key, value):
        """Backing tagged add."""
        self._data.append([tag, key, value])

    def add_user(self, username,  macs,  password):
        """Add a user definition."""
        if username in self._users:
            raise Exception("{} already defined".format(username))
        self._users.append(username)
        for m in macs:
            self._add(self.umac, username, m)
        self._add(self.pwd, username, password)

    def add_mab(self, mac, vlan):
        """Add a MAB."""
        if mac in self._macs:
            raise Exception("{} already defined".format(mac))
        self._macs.append(mac)
        self._add(self.mac, mac, vlan)

    def add_audit(self, user, objs):
        """Add an audit entry."""
        self._add(self.audit, user, objs)

    def get_eap_mab(self):
        """Get eap entries for MAB."""
        for m in self.get_tag(self.mac):
            v = m[1]
            if not isinstance(v, int):
                v = self._get_vlan(v)
            yield [m[0], v]

    def get_eap_user(self):
        """Get eap users."""
        for u in self.get_tag(self.pwd):
            if u[0] in self._mab:
                continue
            vlan = u[0].split(".")[0]
            yield [u[0], u[1], self._get_vlan(vlan)]

    def _get_vlan(self, name):
        """Get vlans."""
        return self._vlans[name]

    def get_vlan_names(self):
        """Return vlan names (used, not all)."""
        return self._vlans.keys()


def main():
    """main entry."""
    success = False
    try:
        parser = argparse.ArgumentParser()
        parser.add_argument("--output", type=str, default="bin/")
        args = parser.parse_args()
        _process(args.output)
        success = True
    except Exception as e:
        print('unable to compose')
        print(str(e))
    if success:
        print("success")
        exit(0)
    else:
        print("failure")
        exit(1)


if __name__ == "__main__":
    main()
