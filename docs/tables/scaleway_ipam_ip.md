---
title: "Steampipe Table: scaleway_ipam_ip - Query Scaleway IPAM IPs using SQL"
description: "Allows users to query the IP addresses managed by Scaleway IP Address Manager (IPAM), including the private IPs of Instances on Private Networks and the resources each IP is attached to."
---

# Table: scaleway_ipam_ip - Query Scaleway IPAM IPs using SQL

Scaleway IP Address Manager (IPAM) reserves and tracks the IP addresses of Scaleway products. Every IP a resource gets on a Private Network, such as the private IP of an Instance, a Load Balancer or a Managed Database, is an IPAM IP, and so are the routed public IPs of Instances. Each one records the pool it was reserved in and the resource it is attached to.

IPAM IPs are regional resources.

## Table Usage Guide

The `scaleway_ipam_ip` table provides insights into the IP addresses allocated in your Scaleway Projects. As a network engineer, use it to find the private IP of a resource, to see which addresses of a Private Network are in use and by what, or to find reserved IPs that are no longer attached to anything.

The Instance API does not return the IPs of a server's private NICs, only their `ipam_ip_ids`, so this table is where those addresses come from.

## Examples

### Basic info
Explore the IPs reserved in your Projects and what they are attached to.

```sql+postgres
select
  id,
  address,
  is_ipv6,
  resource_type,
  resource_name,
  region,
  project
from
  scaleway_ipam_ip;
```

```sql+sqlite
select
  id,
  address,
  is_ipv6,
  resource_type,
  resource_name,
  region,
  project
from
  scaleway_ipam_ip;
```

### Get the private IPs of each Instance
Find the address each Instance has on its Private Networks, for instance to reach it through a bastion or a VPN.

```sql+postgres
select
  s.name as server_name,
  nic ->> 'private_network_id' as private_network_id,
  host(i.address) as private_ip
from
  scaleway_instance_server as s,
  jsonb_array_elements(s.private_nics) as nic
  join scaleway_ipam_ip as i on i.resource_id = nic ->> 'id'
where
  i.resource_type = 'instance_private_nic'
  and not i.is_ipv6;
```

```sql+sqlite
select
  s.name as server_name,
  json_extract(nic.value, '$.private_network_id') as private_network_id,
  i.address as private_ip
from
  scaleway_instance_server as s,
  json_each(s.private_nics) as nic
  join scaleway_ipam_ip as i on i.resource_id = json_extract(nic.value, '$.id')
where
  i.resource_type = 'instance_private_nic'
  and i.is_ipv6 = 0;
```

### List the IPs in use on a Private Network
Determine which addresses of a Private Network are taken and by which resource, which helps when planning static allocations.

```sql+postgres
select
  address,
  resource_type,
  resource_name,
  resource_mac_address
from
  scaleway_ipam_ip
where
  private_network_id = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890';
```

```sql+sqlite
select
  address,
  resource_type,
  resource_name,
  resource_mac_address
from
  scaleway_ipam_ip
where
  private_network_id = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890';
```

### List reserved IPs not attached to any resource
Identify IPs that were reserved but are attached to nothing, which may be leftovers worth releasing.

```sql+postgres
select
  id,
  address,
  private_network_id,
  created_at,
  region,
  project
from
  scaleway_ipam_ip
where
  resource_id is null;
```

```sql+sqlite
select
  id,
  address,
  private_network_id,
  created_at,
  region,
  project
from
  scaleway_ipam_ip
where
  resource_id is null;
```
