# trust-manager-csi-driver

> [!WARNING]  
> This project in in POC stage and should not be used for anything you care about

trust-manager-csi-driver is a CSI driver that can be used to mount bundles generated by [trust-manager](https://github.com/cert-manager/trust-manager) into a Pod. 

It can generate volumes that conform to the `openssl rehash` layout produced by the `update-ca-certificates` command:

```
root@example:/# mount | grep /etc/ssl/certs
tmpfs on /etc/ssl/certs type tmpfs (ro,relatime,inode64)
root@example:/# ls /etc/ssl/certs/
02265526.0  1636090b.0	40193066.0  5e98733a.0	773e07ad.0  b0e59380.0		 d4dae3dd.0  ef954a4e.0
03179a64.0  18856ac4.0	4042bcee.0  5f15c80c.0	7aaf71c0.0  b1159c4c.0		 d6325660.0  f081611a.0
062cdee6.0  1d3472b9.0	40547a79.0  5f618aec.0	7f3d5d1d.0  b66938e9.0		 d7e8dc79.0  f0c70a8d.0
064e0aa9.0  1e08bfd1.0	406c9bb1.0  607986c7.0	8160b96c.0  b727005e.0		 d853d49e.0  f249de83.0
06dc52d5.0  1e09d511.0	4304c5e5.0  626dceaf.0	8cb5ee0f.0  b7a5b843.0		 d887a5bb.0  f30dd6ad.0
080911ac.0  244b5494.0	48bec511.0  653b494a.0	8d86cdd1.0  bf53fb88.0		 dc4d6a89.0  f3377b1b.0
09789157.0  2923b3f9.0	4a6481c9.0  68dd7389.0	8d89cda1.0  c01cdfa2.0		 dd8e9d41.0  f387163d.0
0a775a30.0  2ae6433e.0	4b718d9b.0  6b99d060.0	930ac5d2.0  c01eb047.0		 de6d66f3.0  f39fc864.0
0b1b94ef.0  2b349938.0	4bfab552.0  6d41d539.0	93bc0acc.0  c28a8a30.0		 e113c810.0  f51bb24c.0
0bf05006.0  2e5ac55d.0	4f316efb.0  6fa5da56.0	988a38cb.0  c47d9980.0		 e18bfb83.0  fc5a8f99.0
0c4c9b6c.0  32888f65.0	5443e9e3.0  706f604c.0	9b5697b0.0  ca-certificates.crt  e36a6752.0  fe8a2cd8.0
0f5dc4f3.0  349f2832.0	54657681.0  72909395.0	9c2e7d30.0  ca6e4ad9.0		 e69e0fc3.0  ff34af3f.0
0f6fa695.0  3513523f.0	57bcb2da.0  749e9e03.0	9c8dfbd4.0  cbf06781.0		 e73d606e.0
1001acf7.0  3bde41ac.0	5a4d6896.0  75d1b2ed.0	9d04f354.0  cc450945.0		 e868b802.0
106f3e4d.0  3e44d2f7.0	5ad8a5d6.0  76cb8f92.0	a3418fda.0  cd58d51e.0		 e8de2f56.0
116bf586.0  3e45d192.0	5cd81ad7.0  76faf6c0.0	a94d09e5.0  cd8c0d63.0		 ee64a828.0
14bc7599.0  3fb36b73.0	5d3033c5.0  7719f463.0	aee5f10d.0  ce5e74ef.0		 eed8c118.0
```