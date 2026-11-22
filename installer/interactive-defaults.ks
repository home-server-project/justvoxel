bootc --source-imgref registry:ghcr.io/home-server-project/justvoxel-hws:stable --target-imgref ghcr.io/home-server-project/justvoxel-hws:stable

# The known voxel/voxel credential is installer bootstrap only and must be
# changed on first authentication.
rootpw --lock --iscrypted $6$0LPXGhsb5MOJ160J$K97FKtUs0SgIXum7OpGk69m.asoBR0Os86DpbUnes5JO27aV5XzfEgtpVb/Ff8RMKl.izzSgbqwK8XPpYRFV20
user --name=voxel --groups=wheel --password=$6$0LPXGhsb5MOJ160J$K97FKtUs0SgIXum7OpGk69m.asoBR0Os86DpbUnes5JO27aV5XzfEgtpVb/Ff8RMKl.izzSgbqwK8XPpYRFV20 --iscrypted

%post --erroronfail
chage -d 0 voxel
%end
