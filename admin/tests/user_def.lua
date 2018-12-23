network:Tag("xyz")
object = network:Define("d1", "test1")
object.Macs = {VALID_MAC}
object:Assigned(PROD)
network:Untag()
object:Assigned(DEV)

obj2 = network:Define("d2", "test2")
obj2.Macs = {"ffffffffaaaa"}
network:Assign(PROD, {obj2})
