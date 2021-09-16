# Xcluster ovl - Meridio

This overlay forces the routers to join the kubernetes cluster,
while also tainting them. That way only resources with proper
toleration can be scheduled on them.  
The aim is to deploy via Helm:
- two gateway PODs onto vm-201/202:
  These PODs will provide gateway router functionality for Meridio.
  They also get a secondary interface installed that will connect
  them to Meridio (through Multus). While a third interface connects
  them with a TG POD.
- traffic generator POD:
  Connects with the two gateway PODs through a secondary vlan interface
  installed via Multus. The benefit of a TG POD is to properly test Meridio
  when there are more than 1 external gateways available.
  TG also runs BIRD in a separate container to learn and install VIP related routes
  (via BGP). Thus enabling on-the-fly VIP changes being propageted to the TG a well. 


## Basic Usage

Prerequisites:
- environment for starting `xcluster` is setup.
- docker image for the gateways is built/available; refer to gateway directory
  (Usage of local private docker registry is advised because of this.)

To setup the environment source the `Envsettings.k8s` file;

Note: Assign extra memory to VMs. Otherwise you risk running into out-of-memory
issues, where usually one of the VPP forwarder gets killed by the OS.

Refer to [Meridio description](https://github.com/Nordix/Meridio/blob/master/docs/demo/xcluster.md) for details. 

