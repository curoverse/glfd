var q = {"tilepath":763, "allele": [[0,1,0],[0,0,0]], "loq_info":[[[],[],[]],[[],[],[]]], "start_tilestep":0};
var r = tiletogvcf(JSON.stringify(q));
var r_json = JSON.parse(r);
glfd_return(r_json, "  ");
