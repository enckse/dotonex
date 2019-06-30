object = network:Define("2", "test1")
object.Macs = {"112233445566"}
object:Mabed(DEV)

object = network:Define("2", "test1")
object.Macs = {VALID_MAC}
object:Assigned(DEV)
object:Assigned(PROD)

object = network:Define("2", "test1")
object.Macs = {"123456789012"}
object:Mabed(PROD)
