@0xff1efbacb50003cb;
using Go = import "go.capnp";
$Go.package("packet");
##$Go.import("testpkg");


struct PelicanPacketCapn { 
   requestSer    @0:   Int64; 
   responseSer   @1:   Int64; 
   paysize       @2:   Int64; 
   requestAbTm   @3:   Int64; 
   requestLpTm   @4:   Int64; 
   responseLpTm  @5:   Int64; 
   responseAbTm  @6:   Int64; 
   key           @7:   Text; 
   paymac        @8:   List(UInt8); 
   payload       @9:   List(UInt8); 
} 
#capnp compile -ogo packet.capnp

