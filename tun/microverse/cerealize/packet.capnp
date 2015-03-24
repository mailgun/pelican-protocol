@0xcb94a059ac955bad;
using Go = import "go.capnp";
$Go.package("main");
$Go.import("testpkg");


struct PelicanPacketCapn { 
   responseSerial  @0:   Int64; 
   requestSerial   @1:   Int64; 
   key             @2:   Text; 
   mac             @3:   List(UInt8); 
   payload         @4:   List(UInt8); 
   requestAbTm     @5:   Int64; 
   requestLpTm     @6:   Int64; 
   responseLpTm    @7:   Int64; 
   responseAbTm    @8:   Int64; 
} 

##compile with:

##
##
##   capnp compile -ogo odir/schema.capnp

