package main

// Raw Panel ASCII Text example strings
var HWCtextStrings = []string{

	// Basic examples, mixed:
	"HWCt#0=|||||START",                              // Header
	"HWCt#0=||||1|Basics|One|||||||||1|9|4",          // Two lines
	"HWCt#0=",                                        // Blank out display with empty string
	"HWCt#0=32767",                                   // 16 bit integer
	"HWCt#0=-9999",                                   // 16 bit integer, negative.
	"HWCt#0=32767|1||Float2",                         // Float with 2 decimal points ("32.77")
	"HWCt#0=299|2||Percent",                          // Integer value in Percent
	"HWCt#0=999|3||dB",                               // Integer value in dB
	"HWCt#0=1234|4||Frames",                          // Integer in frames
	"HWCt#0=999|5||Reciproc",                         // Reciproc value of integer
	"HWCt#0=9999|6||Kelvin",                          // Kelvin
	"HWCt#0=9999|7||[Empty!]",                        // format 7 = empty!
	"HWCt#0=-3276|8||Float3",                         // Float with 3 decimal points, optimized for 5 char wide space. Op to +/-9999. "-3.276"
	"HWCt#0=-3276|9||Float2",                         // Float with 3 decimal points, optimized for 5 char wide space. Op to +/-9999. "-32.76"
	"HWCt#0=-276|9||Float2",                          // Float with 3 decimal points, optimized for 5 char wide space. Op to +/-9999. "-2.76"
	"HWCt#0=||1|[Fine]|1",                            // Fine marker set, title as "label"
	"HWCt#0=||1|Title String",                        // no value, just title string (and with "fine" indicator)
	"HWCt#0=|||Title String|1",                       // Title string as label (no "bar" in title)
	"HWCt#0=|||Title string|1|Text1Label",            // Text1label - tall font
	"HWCt#0=|||Title string|1|Text1Label||0",         // Adding the zero (value 2) means we will print two lines and the text label will be in smaller printing
	"HWCt#0=|||Title string|1|Text1Label|Text2Label", // Printing two text lines - automatically the size is reduced
	"HWCt#0=|||Title string|1||Text2Label",           // Printing only the second line - automatically the size is reduced
	"HWCt#0=123|||Title string|1|Val1:|Val2:|456",    // First and second value is printed in small characters with prefix labels Val1 and Val2

	"HWCt#0=1|11||||TextLabel1|TextLabel2", // Printing two labels - Small
	"HWCt#0=2|11||||Text1|Text2",           // Printing two labels - Large
	"HWCt#0=1|10||ABCDEFGHIJ",              // Single line, size 1
	"HWCt#0=2|10||ABCDE",                   // Single line, size 2
	"HWCt#0=3|10||ABC",                     // Single line, size 3
	"HWCt#0=4|10||AB",                      // Single line, size 4
	"HWCt#0=4|10||99",                      // Single line, size 4

	"HWCt#0=-1234|1||Coords:||x:|y:|4567|2", // box type 2, top
	"HWCt#0=-1234|1||Coords:||x:|y:|4567|3", // box type 3, bottom
	"HWCt#0=-1234|1||Coords:||x:|y:|4567|4", // box type 4 both

	"HWCt#0=-500|1||Coords:||||||1|-1000|1000|-700|700", // 1=fill scale
	"HWCt#0=-500|1||Coords:||||||2|-1000|1000|-700|700", // 2=other scale type

	"HWCt#0=||||1|Basic|Two|||||||||1|9|4", // Header
	"HWCt#0=12345|||No format",             // 32 bit integer
	"HWCt#0=-1234567|||No format",          // 32 bit integer, negative
	"HWCt#0=9999|7||Format=7",              // format 7 = empty!
	"HWCt#0=99|2||Format=2",                // Integer value in Percent
	"HWCt#0=12345|2||Format=2",             // Integer value in Percent
	"HWCt#0=999|3||Format=3",               // Integer value in dB
	"HWCt#0=12345|3||Format=3",             // Integer value in dB
	"HWCt#0=1234|4||Format=4",              // Integer in frames
	"HWCt#0=999|5||Format=5",               // Reciproc value of integer
	"HWCt#0=9999|6||Format=6",              // Kelvin

	// Testing icons:
	"HWCt#0=||||1|Test|Icons|||||||||1|9|4", // Header
	"HWCt#0=9999|2|1|Icon=1",                // "Fine" icon
	"HWCt#0=9999|2|2|Icon=2",                // Lock icon
	"HWCt#0=9999|2|3|Icon=3",                // No access
	"HWCt#0=9999|2|8|C.Icon=1",              // Cycle
	"HWCt#0=9999|2|16|C.Icon=2",             // Down
	"HWCt#0=9999|2|24|C.Icon=3",             // Up
	"HWCt#0=9999|2|32|C.Icon=4",             // Hold
	"HWCt#0=9999|2|40|C.Icon=5",             // Toggle
	"HWCt#0=9999|2|48|C.Icon=6",             // OK
	"HWCt#0=9999|2|56|C.Icon=7",             // Question

	// Floating:
	"HWCt#0=||||1|Floating|Point|||||||||1|9|4", // Header
	"HWCt#0=-50|1||Format=1",                    // Float with 2 decimal points. Renders as -0.05
	"HWCt#0=-550|1||Format=1",                   // Float with 2 decimal points. Renders as -0.56
	"HWCt#0=-5550|1||Format=1",                  // Float with 2 decimal points.  Renders as -5.55
	"HWCt#0=-55550|1||Format=1",                 // Float with 2 decimal points.  Renders as -55.56
	"HWCt#0=-5|8||Format=8",                     // Float with 3 decimal points. Renders as -0.005
	"HWCt#0=-55|8||Format=8",                    // Float with 3 decimal points. Renders as -0.055
	"HWCt#0=-555|8||Format=8",                   // Float with 3 decimal points. Renders as -0.555
	"HWCt#0=-5555|8||Format=8",                  // Float with 3 decimal points. Renders as -5.555
	"HWCt#0=-55555|8||Format=8",                 // Float with 3 decimal points. Renders as -55.555
	"HWCt#0=-5|9||Format=9",                     // Float with 2 decimal points. Renders as -0.05
	"HWCt#0=-55|9||Format=9",                    // Float with 2 decimal points. Renders as -0.55
	"HWCt#0=-555|9||Format=9",                   // Float with 2 decimal points. Renders as -5.55
	"HWCt#0=-5555|9||Format=9",                  // Float with 2 decimal points. Renders as -55.55
	"HWCt#0=-55555|9||Format=9",                 // Float with 2 decimal points. Renders as -555.55
	"HWCt#0=-5|12||Format=12",                   // Float with 1 decimal points. Renders as -0.5
	"HWCt#0=-55|12||Format=12",                  // Float with 1 decimal points. Renders as -5.5
	"HWCt#0=-555|12||Format=12",                 // Float with 1 decimal points. Renders as -55.5
	"HWCt#0=-5555|12||Format=12",                // Float with 1 decimal points. Renders as -555.5
	"HWCt#0=-55555|12||Format=12",               // Float with 1 decimal points. Renders as -5555.5

	// Title Bar:
	"HWCt#0=||||1|Title|Bar|||||||||1|9|4",                // Header
	"HWCt#0=|||Bar = Value",                               // Title string as value (has a solid "bar" in title)
	"HWCt#0=|||Line = Label|1",                            // Title string as label (has a line under title string)
	"HWCt#0=|||My Title|1|Font 1 8x8|As Label|||||||||8",  // Title Font test
	"HWCt#0=|||My Title||Font 1 8x8|As Value|||||||||8",   // Title Font test
	"HWCt#0=|||My Title|1|Font 2 5x5|As Label|||||||||16", // Title Font test
	"HWCt#0=|||My Title||Font 2 5x5|As Value|||||||||16",  // Title Font test
	"HWCt#0=|||My Title||Font 1 8x8|Large|||||||||8|32",   // Title Font Size test (wide font)
	"HWCt#0=|||My Title||Font 1 8x8|Large|||||||||8|128",  // Title Font Size test (tall font)
	"HWCt#0=|||My Title||Font 1 8x8|Large|||||||||8|160",  // Title Font Size test (double size font)

	// Text Labels
	"HWCt#0=||||1|Text|Labels|||||||||1|9|4",                 // Header
	"HWCt#0=|||Short Text Label|1|Quick",                     // Typical Text Label
	"HWCt#0=|||Medium Text Label|1|Quick Dog",                // Typical Text Label
	"HWCt#0=|||Long Text Label|1|Quick Dog Lazy Fox",         // Typical Text Label
	"HWCt#0=|||Small Font|1|Quick Dog Lazy Fox|||||||||||5",  // Small text
	"HWCt#0=|||Narrow Font|1|Quick Dog Lazy Fox|||||||||||9", // Narrow text
	"HWCt#0=|||One Label Line|1|Text1Label||0",               // Adding the zero (value 2) means we will print two lines and the text label will be in smaller printing
	"HWCt#0=|||Two Label Lines|1|Text1Label|Text2Label",      // Printing two labels
	"HWCt#0=|||Only Second Line|1||Text2Label",               // Printing only the second line
	"HWCt#0=123|||Text & Value|1|Val1:|Val2:|456",            // First and second value is printed in small characters with prefix labels Val1 and Val2

	// Special text
	"HWCt#0=||||1|Special|Text|||||||||1|9|4",                               // Header
	"HWCt#0=1|11||||The quick brown fox|jumps over the lazy dog.",           // Printing two labels, size 1
	"HWCt#0=2|11||||The quick brown fox|jumps over the lazy dog.",           // Printing two labels, size 2
	"HWCt#0=|11||||The quick brown fox|jumps over the lazy dog.||||||||||9", // Printing two labels, size 1 tall
	"HWCt#0=1|10||Quick brown fox jumps over the lazy dog.",                 // Printing one label, size 1
	"HWCt#0=2|10||Quick brown fox jumps over the lazy dog.",                 // Printing one label, size 2
	"HWCt#0=3|10||Quick brown fox jumps over the lazy dog.",                 // Printing one label, size 3
	"HWCt#0=4|10||Quick brown fox jumps over the lazy dog.",                 // Printing one label, size 4
	"HWCt#0=4|10||12345", // Printing one label, size 4
	"HWCt#0=|11||||The quick brown fox|Font 1|||||||||1",             // Another font
	"HWCt#0=|11||||The quick brown fox|+Fixed Width|||||||||65",      // Another font, fixed width
	"HWCt#0=|11||||The quick brown fox|Taller|||||||||1|9",           // Taller
	"HWCt#0=|11||||The quick brown fox|+ char spacing|||||||||1|5|8", // Extra character spacing

	// Pair of Coordinates
	"HWCt#0=||||1|Pair of|Coordinates|||||||||1|9|4", // Header
	//	"HWCt#0=-1234|1||No Pair||x:|y:|4567|0",          // No box - INCOMPATIBLE (and accepted as such). Should not add the second title if you don't want a pair, since second title will automatically activate the pairing mode = 1
	"HWCt#0=-1234|1||No Box:||x:|y:|4567|1",     // Box type 1
	"HWCt#0=-1234|1||Box Upper:||x:|y:|4567|2",  // Box type 2
	"HWCt#0=-1234|1||Box Lower:||x:|y:|4567|3",  // Box type 3
	"HWCt#0=-1234|1||Both Boxed:||x:|y:|4567|4", // Box type 4

	// 1 = strength scale
	"HWCt#0=||||1|Scale|Bar|||||||||1|9|4", // Header
	"HWCt#0=-2000|1||Scale 1||||||1|-2000|1000|-2000|1000",
	"HWCt#0=-1000|1||Scale 1||||||1|-2000|1000|-2000|1000",
	"HWCt#0=0|1||Scale 1||||||1|-2000|1000|-2000|1000",
	"HWCt#0=10|1||Scale 1||||||1|-2000|1000|-2000|1000",
	"HWCt#0=1000|1||Scale 1||||||1|-2000|1000|-2000|1000",
	"HWCt#0=-550|1||Scale 1+L||||||1|-2000|1000|-1800|600",

	// 2 = centered marker scale
	"HWCt#0=-2000|1||Scale 2||||||2|-2000|1000|-2000|1000",
	"HWCt#0=-1000|1||Scale 2||||||2|-2000|1000|-2000|1000",
	"HWCt#0=0|1||Scale 2||||||2|-2000|1000|-2000|1000",
	"HWCt#0=10|1||Scale 2||||||2|-2000|1000|-2000|1000",
	"HWCt#0=1000|1||Scale 2||||||2|-2000|1000|-2000|1000",
	"HWCt#0=-550|1||Scale 2+L||||||2|-2000|1000|-1800|600",

	// 3 = centered bar
	"HWCt#0=-2000|1||Scale 3||||||3|-2000|1000|-2000|1000",
	"HWCt#0=-1000|1||Scale 3||||||3|-2000|1000|-2000|1000",
	"HWCt#0=0|1||Scale 3||||||3|-2000|1000|-2000|1000",
	"HWCt#0=10|1||Scale 3||||||3|-2000|1000|-2000|1000",
	"HWCt#0=1000|1||Scale 3||||||3|-2000|1000|-2000|1000",
	"HWCt#0=-500|1||Scale 3+L||||||3|-2000|1000|-1800|600",

	"HWCt#0=-1234|1||Inverted||x:|y:|4567|2||||||||||1", // Box type 2

	"HWCt#0=||||1|Color|Images|||||||||1|9|4", // Header
	`{"HWCIDs":[38],"HWCMode": {"State": 4},"HWCColor": {"ColorIndex": {"Index": 9}},"HWCText": {"IntegerValue": 9999,"Formatting": 2,"ModifierIcon": 5,"Title": "Value:","SolidHeaderBar": true}}`, // JSON example of state
}

var HWCgfxStrings = [][]string{
	{ // "D" in "DUMB"
		"HWCg#0=0:///////gAAD///////+AAP////////AA/////////gD/////////gP/////////A//////////D///gAP///+P//+AAD///4///4AAD///z///gAAH8=",
		"HWCg#0=1://7///gAAD///v//+AAAP//////4AAAf//////gAAB//////+AAAH//////4AAAf//////gAAB//////+AAAH//////4AAAf//////gAAD///v//+AA=",
		"HWCg#0=2:AD///v//+AAAf//8///4AAH///z///gAD///+P/////////w/////////8D/////////gP////////4A////////8AD///////8AAP//////gAAA",
	},
	{ // TEST 64x48
		"HWCg#0=0/4,64x48://///////////////////8hCEIQhCEITyEIQhCEIQhP//////////8hCEIQhCEITyEIQhCEIQhPIQhCEIQhCE8hCEIQhCEIT//////////8=",
		"HWCg#0=1:yEIQhCEIQhPIQhCEIQhCE8hCEIQhCEITyEIQhCEIQhP//////////8hCEIQhCEITyEIQhCEIQhPIABCEIQACE8gAAAQgAAIT/H4DAHDh8f8=",
		"HWCg#0=2:yP4HAADjuBPIwA8AAeMYE8jAHxzD4xgTyPwbD4bjuBP4/jMPhuHw/8jGdwcM47gTyMZ/hw/zGBPIxgcPgOMYE8juAx3A47gT/HwDGMDh8f8=",
		"HWCg#0=3:yAAAAAAAAhPIABAAAQACE8hCEIQhCEITyEIQhCEIQhP//////////8hCEIQhCEITyEIQhCEIQhPIQhCEIQhCE8hCEIQhCEIT//////////8=",
		"HWCg#0=4:yEIQhCEIQhPIQhCEIQhCE8hCEIQhCEITyEIQhCEIQhP//////////8hCEIQhCEIT/////////////////////w==",
	},
	{
		// TEST 96x48
		"HWCg#0=0/7,96x48:////////////////////////////////xCEIQhCEIQhCEIQjxCEIQhCEIQhCEIQj////////////////xCEIQhCEIQhCEIQjxCEIQhCEIQg=",
		"HWCg#0=1:QhCEI8QhCEIQhCEIQhCEI8QhCEIQhCEIQhCEI////////////////8QhCEIQhCEIQhCEI8QhCEIQhCEIQhCEI8QhCEIQhCEIQhCEI8QhCEI=",
		"HWCg#0=2:EIQhCEIQhCP////////////////EIQhCEIQhCEIQhCPEIQhCEIQhCEIQhCPEIQgAAAQhAAIQhCPEIQgAAAQgAAIQhCP///x8HwBw4fH///8=",
		"HWCg#0=3:xCEI7j8AAOO4EIQjxCEIxnAAAeMYEIQjxCEIxmAcw+MYEIQjxCEIxn4PhuO4EIQj///4/n8PhuHw////xCEIfmOHDOO4EIQjxCEIBmGHD/M=",
		"HWCg#0=4:GBCEI8QhCAZzj4DjGBCEI8QhCP4/HcDjuBCEI////vwfGMDh8f///8QhCAAAAAAAAhCEI8QhCAAAAAEAAhCEI8QhCEIQhCEIQhCEI8QhCEI=",
		"HWCg#0=5:EIQhCEIQhCP////////////////EIQhCEIQhCEIQhCPEIQhCEIQhCEIQhCPEIQhCEIQhCEIQhCPEIQhCEIQhCEIQhCP///////////////8=",
		"HWCg#0=6:xCEIQhCEIQhCEIQjxCEIQhCEIQhCEIQjxCEIQhCEIQhCEIQjxCEIQhCEIQhCEIQj////////////////xCEIQhCEIQhCEIQj//////////8=",
		"HWCg#0=7://///////////////////w==",
	},
	{
		// TEST 64x38
		"HWCg#0=0/15,64x38://///////////////////8QhCA==",
		"HWCg#0=1:QhCEIQvEIQhCEIQhC////////w==",
		"HWCg#0=2:///EIQhCEIQhC8QhCEIQhCELxA==",
		"HWCg#0=3:IQhCEIQhC8QhCEIQhCEL/////w==",
		"HWCg#0=4://///8QhCEIQhCELxCEIQhCEIQ==",
		"HWCg#0=5:C8QBCEIQAAELxAAAAgAAAQv8fg==",
		"HWCg#0=6:AwAHwfH/xP4HAA/juQvAwA8AAA==",
		"HWCg#0=7:YxgLwMAfHMBjGAvA/BsPgOO4Cw==",
		"HWCg#0=8:+P4zD4fB8P/AxncHAGO4C8DGfw==",
		"HWCg#0=9:hwBjGAvAxgcPgGMYC8TuAx3P4w==",
		"HWCg#0=10:uQv8fAMYz8Hx/8QAAAAAAAELxA==",
		"HWCg#0=11:AQgAAAABC8QhCEIQhCELxCEIQg==",
		"HWCg#0=12:EIQhC///////////xCEIQhCEIQ==",
		"HWCg#0=13:C8QhCEIQhCELxCEIQhCEIQvEIQ==",
		"HWCg#0=14:CEIQhCEL///////////EIQhCEA==",
		"HWCg#0=15:hCEL/////////////////////w==",
	},
	{
		// TEST 112x32
		"HWCg#0=0/5,112x32://///////////////////////////////////8QhCEIQhCEIQhCEIQhDxCEIQhCEIQhCEIQhCEPEIQhCEIQhCEIQhCEIQ////////////w==",
		"HWCg#0=1:///////EIQhCEIQhCEIQhCEIQ8QhCEIQhCEIQhCEIQhDxCEIQhCEIQhCEIQhCEPEIQhAEAQBCAAABCEIQ////+AAAAD+AAAf////xCEIQw==",
		"HWCg#0=2:AcH4AH4fhCEIQ8QhCE8Hw/wAfx/EIQhDxCEITwfAHAADAcQhCEPEIQhDAcAczgMBxCEIQ////8MBwDj8BwOP////xCEIQwHA8Hg+DwQhCA==",
		"HWCg#0=3:Q8QhCEMBwcA4Bw4EIQhDxCEIQwHBgHgDGAQhCEPEIQhDAcOAfAMYBCEIQ////9/H8/jOfx/f////xCEIT8fz+cZ+H8QhCEPEIQhAAAAAAA==",
		"HWCg#0=4:AAAEIQhDxCEIQAAAAAAAAAQhCEPEIQhCEAQBCEIQhCEIQ///////////////////xCEIQhCEIQhCEIQhCEPEIQhCEIQhCEIQhCEIQ8QhCA==",
		"HWCg#0=5:QhCEIQhCEIQhCEPEIQhCEIQhCEIQhCEIQ/////////////////////////////////////8=",
	},
	{
		// TEST 128x32
		"HWCg#0=0/9,128x32:///////////////////////////////////////////EIQhCEIQhCEIQhCEIQhCHxCEIQhCEIQg=",
		"HWCg#0=1:QhCEIQhCEIfEIQhCEIQhCEIQhCEIQhCH/////////////////////8QhCEIQhCEIQhCEIQhCEIc=",
		"HWCg#0=2:xCEIQhCEIQhCEIQhCEIQh8QhCEIQhCEIQhCAIQhCEIfEIQhCEAAAAEIAACEIQhCH/////+AAAAA=",
		"HWCg#0=3:/gAAH//////EIQhCAwfh+AB+H4EIQhCHxCEIQh8H8/wAfx/BCEIQh8QhCEIfAHOcAAMBwQhCEIc=",
		"HWCg#0=4:xCEIQgMAc5zOAwHBCEIQh//////DAOH4/AcDj//////EIQhCAwPB+Hg+DwEIQhCHxCEIQgMDg5w=",
		"HWCg#0=5:OAcOAQhCEIfEIQhCAwYDDHgDGAEIQhCHxCEIQgMGA5x8AxgBCEIQh//////fx/P8zn8f3/////8=",
		"HWCg#0=6:xCEIQh/H8fnGfh/BCEIQh8QhCEIAAAAAAAAAIQhCEIfEIQhCEAAAAAAAACEIQhCHxCEIQhCEIQg=",
		"HWCg#0=7:QgCEIQhCEIf/////////////////////xCEIQhCEIQhCEIQhCEIQh8QhCEIQhCEIQhCEIQhCEIc=",
		"HWCg#0=8:xCEIQhCEIQhCEIQhCEIQh8QhCEIQhCEIQhCEIQhCEIf///////////////////////////////8=",
		"HWCg#0=9://////////8=",
	},
	{
		// TEST 48x24
		"HWCg#0=0/2,48x24:////////////////0IQhCEIT0IQhCEIT0IQhCEIT/AAf+A+f0AABAAADw4/AB+Bzw4zABvBzx5g=",
		"HWCg#0=1:bmAw88+MxuBh89mPw8HjM9mNw4ODM//c44MH+//c58YH+8GPxufgM8GHjGfgM9AAAAAAA9AAAAA=",
		"HWCg#0=2:AhPQhCEIQhP////////QhCEIQhP///////////////8=",
	},
	{
		// TEST 52x24  (doesn't work on bin panels?)
		"HWCg#0=0/2,52x24:///////////////////EIQhCEIQ/xCEIQhCEP8QhCEIQhD/4AAf+A+f/wAAAQAAAP8fj4AH4HD8=",
		"HWCg#0=1:xwJwAbwcP8YAM5gMPD/nAHG4GHx/x+Dg8HjMP8BhwODgzD/AYwDgwf4/wGMB8YH+P+fj8bn4DP8=",
		"HWCg#0=2:x8PzGfgMP8AAAAAAAD/EAAAAAIQ/xCEIQhCEP//////////EIQhCEIQ///////////////////8=",
	},
	{
		// TEST 256x20
		"HWCg#0=0/7,256x20://///////////////////////////////////////////////////////////////////////////////////////////////////wAAAAA=",
		"HWCg#0=1:IQAAAIQhCEIQhCEIQhCEI4QhCEIQhCEIQhCEIwAAAAAhAAAAhCEIQhCEIQhCEIQjhCEIQhCEIQhCEIQjfwfg/gAP4PwEIQhCEIQhCEIQhCM=",
		"HWCg#0=2:BCEIQhCEIQhCEIQjf4fh/gAP8f5///////////////9///////////////8DjgGAAABxzgQhCEIQhCEIQhCEIwQhCEIQhCEIQhCEIwOOA4A=",
		"HWCg#0=3:cYBxhgQhCEIQhCEIQhCEIwQhCEIQhCEIQhCEIwcPg/g7gPGGBCEIQhCEIQhCEIQjBCEIQhCEIQhCEIQjDg/j/h8B4YYEIQhCEIQhCEIQhCM=",
		"HWCg#0=4:BCEIQhCEIQhCEIQjPADzjh4DgYY///////////////8///////////////84AHOGDgcBhgQhCEIQhCEIQhCEIwQhCEIQhCEIQhCEI3AAc4Y=",
		"HWCg#0=5:Hw4BhgQhCEIQhCEIQhCEIwQhCEIQhCEIQhCEI2AAcY4/jgHOBCEIQhCEIQhCEIQjBCEIQhCEIQhCEIQjf4/h/jOP8f4EIQhCEIQhCEIQhCM=",
		"HWCg#0=6:BCEIQhCEIQhCEIQjf4/A/HHP8Pz///////////////////////////////8AAAAAAAAAAIQhCEIQhCEIQhCEI4QhCEIQhCEIQhCEIwAAAAA=",
		"HWCg#0=7:AAAAAIQhCEIQhCEIQhCEI4QhCEIQhCEIQhCEI/////////////////////////////////////////////////////////////////////8=",
	},
	{
		// TEST 64x32
		"HWCg#0=0/1,64x32://///////////////////8QhCEIQhCELxCEIQhCEIQvEIQhCEIQhC///////////xCEIQhCEIQvEIQhCEIQhC8QBCEIQBCELxAEIQhAAAAv8AOAP8AAAP8D8A4AB/D+LwfwHgAH8P4vDgA+AAA4Bi8OAD45wDgGL8/gbh2AcB5/D/DOH4fwPC8OOc4PA/B4Lw45/w8AOOAvDjn/HwA4wC/OOA4fgDnAfwfw=",
		"HWCg#0=1:A45x/H+LwPgDnHH4f4vEAAAAAAAAC8QBCAAAAAAL//////8f8//EIQhCEIQhC8QhCEIQhCELxCEIQhCEIQvEIQhCEIQhC/////////////////////8=",
	},
	{
		// TEST 48x24    (doesn't work on neither bin nor ascii panels?)
		"HWCg#0=0/0,48x24:////////////////0IQhCEIT0IQhCEIT0IQhCEIT/AAf+A+f0AABAAADw4/AB+Bzw4zABvBzx5huYDDzz4zG4GHz2Y/DweMz2Y3Dg4Mz/9zjgwf7/9znxgf7wY/G5+AzwYeMZ+Az0AAAAAAD0AAAAAIT0IQhCEIT////////0IQhCEIT////////////////",
	},
	{ // graywedge.png as Gray  (doesn't work on bin panels?)
		"HWCgGray#0=0/6,64x32:AAAAERESIiMzNERVVmZ3eIiJmaqqu7vMzN3d7u7v//8AAAARERIiIzM0RFVWZnd4iImZqqq7u8zM3d3u7u///wAAABEREiIjMzREVVZmd3iIiZmqqru7zMzd3e7u7///AAAAERESIiMzNERVVmZ3eIiJmaqqu7vMzN3d7u7v//8AAAARERIiIzM0RFVWZnd4iImZqqq7u8zM3d3u7u///wAAABEREiIj",
		"HWCgGray#0=1:MzREVVZmd3iIiZmqqru7zMzd3e7u7///AAAAERESIiMzNERVVmZ3eIiJmaqqu7vMzN3d7u7v//8AAAARERIiIzM0RFVWZnd4iImZqqq7u8zM3d3u7u///wAAABEREiIjMzREVVZmd3iIiZmqqru7zMzd3e7u7///AAAAERESIiMzNERVVmZ3eIiJmaqqu7vMzN3d7u7v//8AAAARERIiIzM0RFVWZnd4",
		"HWCgGray#0=2:iImZqqq7u8zM3d3u7u///wAAABEREiIjMzREVVZmd3iIiZmqqru7zMzd3e7u7///AAAAERESIiMzNERVVmZ3eIiJmaqqu7vMzN3d7u7v//8AAAARERIiIzM0RFVWZnd4iImZqqq7u8zM3d3u7u////////////////////////8AAAAAAAAAAAAAAAAAAAAA/////////////////////wAAAAAAAAAA",
		"HWCgGray#0=3:AAAAAAAAAAD/////////////////////AAAAAAAAAAAAAAAAAAAAAP////////////////////8AAAAAAAAAAAAAAAAAAAAA///+7u7d3czMu7uqqpmYiId3ZmVVREMzMiIhEREAAAD///7u7t3dzMy7u6qqmZiIh3dmZVVEQzMyIiEREQAAAP///u7u3d3MzLu7qqqZmIiHd2ZlVURDMzIiIRERAAAA",
		"HWCgGray#0=4:///+7u7d3czMu7uqqpmYiId3ZmVVREMzMiIhEREAAAD///7u7t3dzMy7u6qqmZiIh3dmZVVEQzMyIiEREQAAAP///u7u3d3MzLu7qqqZmIiHd2ZlVURDMzIiIRERAAAA///+7u7d3czMu7uqqpmYiId3ZmVVREMzMiIhEREAAAD///7u7t3dzMy7u6qqmZiIh3dmZVVEQzMyIiEREQAAAP///u7u3d3M",
		"HWCgGray#0=5:zLu7qqqZmIiHd2ZlVURDMzIiIRERAAAA///+7u7d3czMu7uqqpmYiId3ZmVVREMzMiIhEREAAAD///7u7t3dzMy7u6qqmZiIh3dmZVVEQzMyIiEREQAAAP///u7u3d3MzLu7qqqZmIiHd2ZlVURDMzIiIRERAAAA///+7u7d3czMu7uqqpmYiId3ZmVVREMzMiIhEREAAAD///7u7t3dzMy7u6qqmZiI",
		"HWCgGray#0=6:h3dmZVVEQzMyIiEREQAAAA==",
	},
	{ // Color image
		"HWCgRGB#0=0/24,64x32:AB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4Afg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OAAHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAf",
		"HWCgRGB#0=1:AB8H4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+D44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44AAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4Afg",
		"HWCgRGB#0=2:B+AH4Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44PjgAB8AHwAfYt+cv5y/nL+cv5y/nL+cv5y/nL9i3wgfAB8AHwAfAB8AHwAfB+AH4AfgB+AH4AfgB+A34H/gn+Of45/jn+OP4FfgB+AH4AfgB+AH4Afg+OD44Pjg+OD44Pjg/PP88/zz/PP88/zz/PP88/zz/HH66vjg",
		"HWCgRGB#0=3:+OD44Pjg+OAAHwAfAB+cv/////////////////////////////+MPwAfAB8AHwAfAB8H4AfgB+AH4Afgn+Pv+//////////////////////v+5/jB+AH4AfgB+D44Pjg+OD44Pjg+OD//////////////////////////////pr5gfjg+OD44AAfAB8AH5y//////5y/AB8AHwAfAB8AH2Lf////////",
		"HWCgRGB#0=4:AB8AHwAfAB8AHwfgB+AH4Afg3/n/////x/N/4AfgB+AH4Afgj+Dv+/////+f4wfgB+AH4Pjg+OD44Pjg+OD44P////////jg+OD44Pjg+OD5gf44//////zz+OD44PjgAB8AHwAfnL//////71/Wf9Z/1n/Wf9Z/95//////nL8AHwAfAB8AHwAfB+AH4Afgj+D/////x/MH4AfgB+AH4AfgB+AH4Afg",
		"HWCgRGB#0=5:Z+AH4AfgB+AH4Afg+OD44Pjg+OD44Pjg/////////PP88/zz/PP88/3W/77///6a+YH44Pjg+OAAHwAfAB+cv//////vX9Z/1n/////////e33ufML8AHwAfAB8AHwAfAB8H4AfgB+Cf4/////+v6wfgB+AH4Afg////////////////1/YH4AfgB+D44Pjg+OD44Pjg+OD/////////////////////",
		"HWCgRGB#0=6://////77/VX6R/jg+OD44AAfAB8AH5y//////5y/AB8AHwgfrT//////719i3wAfAB8AHwAfAB8AHwfgB+AH4Dfg9/3///f9Z+AH4AfgB+Cf45/jn+O38P/////X9gfgB+AH4Pjg+OD44Pjg+OD44P////////jg+OD44Pjg+OD44PwP//////9d+OD44PjgAB8AHwAfnL//////nL8AHwAfAB8AH3uf",
		"HWCgRGB#0=7:95//////jD8AHwAfAB8AHwAfB+AH4AfgB+Bn4O/7/////9/5r+uf45/jn+PX9t/5/////7fwB+AH4Afg+OD44Pjg+OD44Pjg/////////PP88/zz/PP88/zz/vv//////vv44Pjg+OAAHwAfAB+cv/////+cvwAfAB8AHwAfAB8wv97f/////7W/AB8AHwAfAB8H4AfgB+AH4AfgB+CP4Lfw1/b/////",
		"HWCgRGB#0=8://///9/5x/Of41fgB+AH4AfgB+D44Pjg+OD44Pjg+OD///////////////////////////6a/PP5gfjg+OD44AAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4Pjg+OD44Pjg+OD44Pjg+OD44Pjg",
		"HWCgRGB#0=9:+OD44Pjg+OD44Pjg+OD44Pjg+OD44PjgAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4Afg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OAAHwAfAB8AHwAfAB8AHwAf",
		"HWCgRGB#0=10:AB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8H4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+D44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44AAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwfgB+AH4AfgB+AH4Afg",
		"HWCgRGB#0=11:B+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44PjgAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4Afg+OD44Pjg+OD44Pjg",
		"HWCgRGB#0=12:+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OAAHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8H4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+D44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44AAfAB8AHwAf",
		"HWCgRGB#0=13:AB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwAfAB8AHwfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4AfgB+AH4Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44Pjg+OD44PjgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP//////3/e+953vXecc3vvWms5Z",
		"HWCgRGB#0=14:xhi917WWpTSc85SSjFF78HOua21jDFrLSmlCKDnnMaYpRSEkGOMQoghhCEEAIAAAAAD///////////////////////////////////////8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA///////f9773ne9d5xze+9a6zlnGGL3XtZalNJzzlJKMUXwPc65rbWMMWstKaUIoOecxhilFIQQY4xCi",
		"HWCgRGB#0=15:CGEIQQAgAAAAAP///////////////////////////////////////wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD//////9/3vvee713nHN7b1prOWcYYvde1lqU0nPOUsoxRe+9zrmtNYwxay0ppQig55zGmKUUhJBjjEKIIYghBACAAAAAA////////////////////////////////////////",
		"HWCgRGB#0=16:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP//////3/e+957vXecc3tvWus5Zxhi917WWpTSc85SSjFF8D3Oua01jDFrKSmlCKDnnMYYpZSEkGOMQoghhCEEAIAAAAAD///////////////////////////////////////8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA///////f9773nu9d",
		"HWCgRGB#0=17:5xze29a6zlnGGL3XtZWlNJzzlJKMUXvwc65rbWMMWstKaUIoOecxhillIQQY4xCiCGEIQQAgAAAAAP///////////////////////////////////////wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD//////9/3vvee713nHN7b1rrOWcYYvde1lqU0nPOUkoxRhA9zrmttYwxay0ppQig55zGm",
		"HWCgRGB#0=18:KUUhJBjjEKIIYQhBACAAAAAA////////////////////////////////////////AB8AHxAfQB9gH4gfqB/IH+gf+B/4H/gf+B74GvgW+DL4jfip+MX4wPjg+OD5QPng+sD7oPyg/aD+gP9g/+D/4Ofg1+C34J/gf+Bf4D/gH+AH4AfgB+AH4AfgB+AH4AfgB+kH8gf4B/wH/wd/Br8F3wTfA78CfwC/",
		"HWCgRGB#0=19:AB8AHwAfAB8AHwAfEB9AH2AfiB+oH8gf6B/4H/gf+B/4Hvga+Bb4MviN+Kn4xfjA+OD44PlA+eD6wPug/KD9oP6A/2D/4P/g5+DX4Lfgn+B/4F/gP+AX4AfgB+AH4AfgB+AH4AfgB+AH6QfyB/gH/Af/B38GvwXfBN8DvwJ/AN8AHwAfAB8AHwAfAB8QHzgfYB+IH6gfyB/gH/gf+B/4H/ge+Br4Fvgy",
		"HWCgRGB#0=20:+I34qfjF+MD44Pjg+UD54PrA+6D8oP2g/oD/YP/g/+Dn4Nfgt+Cf4H/gX+A/4BfgB+AH4AfgB+AH4AfgB+AH4AfpB/IH+Af8B/8Hfwa/Bd8E3wO/An8A3wAfAB8AHwAfAB8AHxAfOB9gH4gfqB/IH+gf+B/4H/gf+B74GvgW+DL4jfip+MX4wPjg+OD5QPng+sD7oPyg/aD+gP9g/+D/4Ofg1+C34J/g",
		"HWCgRGB#0=21:f+Bf4D/gF+AH4AfgB+AH4AfgB+AH4AfgB+kH8gf4B/wH/wd/Br8F3wTfA78CfwDfAB8AHwAfAB8AHwAfEB84H2AfiB+oH8gf6B/4H/gf+B/4Hvga+Bb4MviN+Kn4xfjA+OD44PlA+eD6wPug/KD9oP6A/2D/4P/g5+DX4Lfgn+B/4F/gP+Af4AfgB+AH4AfgB+AH4AfgB+AH6QfyB/gH/Af/B38GvwXf",
		"HWCgRGB#0=22:BN8DvwJ/AL8AHwAfAB8AHwAfAB8QHzgfYB+IH6gfyB/oH/gf+B/4H/ge+Br4Fvgy+I34qfjF+MD44Pjg+UD54PrA+6D8oP2g/oD/YP/g/+Dn4Nfgt+Cf4H/gX+A/4BfgB+AH4AfgB+AH4AfgB+AH4AfpB/IH+Af8B/8HfwbfBd8E3wO/An8A3wAfAB8AHwAfAB8AHxAfQB9gH4gfqB/IH+gf+B/4H/gf",
		"HWCgRGB#0=23:+B74GvgW+DL4jfip+MX4wPjg+OD5QPng+sD7oPyg/aD+gP9A/+D/4Ofg1+C34J/gf+Bf4D/gH+AH4AfgB+AH4AfgB+AH4AfgB+kH8gf4B/wH/wd/Br8F3wTfA78CfwDfAB8AHwAfAB8AHwAfEB9AH2AfiB+oH8gf6B/4H/gf+B/4Hvga+Bb4MviN+Kn4xfjA+OD44PlA+eD6wPug/KD9oP6A/2D/4P/g",
		"HWCgRGB#0=24:5+DX4Lfgn+B/4F/gP+AX4AfgB+AH4AfgB+AH4AfgB+AH6QfyB/gH/Af/B38GvwXfBN8DvwJ/AL8AHwAfAB8AHw==",
	},
}
