"""Expired user."""
import netconf as __config__
import users.common as common
normal = __config__.Assignment()
normal.macs = [common.VALID_MAC]
normal.vlan = "dev"
normal.expires = "2017-01-01"
normal.password = 'ddcb5a1547def9aafbd0587e255d8626'
