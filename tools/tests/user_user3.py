"""User with inheritance."""
import netconf as __config__
normal = __config__.Assignment()
normal.macs = ["001122334455"]
normal.vlan = "dev"
normal.owns = ["001122221100"]

admin = __config__.Assignment()
admin.inherits = normal
admin.vlan = "prod"
normal.password = 'e2192da00a1ccba417ec515395a044f7'
