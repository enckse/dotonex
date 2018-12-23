network:Disabled()

object = network:Define("1", "testexpire")
object.Macs = {VALID_MAC}
object:Assigned(DEV)
