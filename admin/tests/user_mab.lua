devices = {}
devices["test1"] = "112233445567"
devices["test2"] = "abcdef123456"
devices["test3"] = "000011112222"
for k, v in pairs(devices) do
    object = network:Define("mabme", k)
    object.Macs = {v}
    object:Mabed(DEV)
end
