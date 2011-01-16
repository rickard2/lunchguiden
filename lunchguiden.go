/* 
  Author: Rickard Andersson - M:rickard@0x539.se - T:rickard2 - W:bennison.se

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>


    This is a simple application for downloading and parsing information from
    a local Swedish lunch menu system called Lunchguiden (http://service.dt.se/lunch/lunch.asp)
    
    It's used as part of a Android application to server clean and formatted
    JSON data to the device, thus taking away the nasty work of parsing the data
    from the Android device. 

    100809: First version ready for use. It took about three days to finish.

*/


package main

import (
	"http"
	"fmt"
	"os"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"flag"
	"crypto/md5"
	"json"
	"bytes"
)

// Three structs needed for JSON output
//
type DataStruct struct {
	City string;
	Week int;
	Days [5]DayData;
}
type DayData struct {
	Day int;
	Name string;
	Restaurants []RestData;
}
type RestData struct {
	Name string;
	ImageUrl string;
	Description string;
	Menu string;
}

// Input values
// 
var url = flag.String("url", "", "URL to lunchguiden");
var out = flag.String("out", "", "Output file");
var city = flag.String("city", "", "Textual representation of the city");
var week = flag.Int("week", 0, "What week number to download");


func main() {
	var (
		res *http.Response;
		err os.Error;
		inData []byte;
	)

	// Textual names of the weekdays, needed for URL generation
	//
	weekdays := []string{ "Mandag", "Tisdag", "Onsdag", "Torsdag", "Fredag" };

	// Parse and validate input 
	//
	flag.Parse();
	
	if *url == "" {
		fmt.Println("ERROR: No URL specified");
		flag.PrintDefaults();
		return;
	}
	if *out == "" {
		fmt.Println("ERROR: No output file specified");
		flag.PrintDefaults();
		return;
	}
	if *city == "" {
		fmt.Println("ERROR: No city specified");
		flag.PrintDefaults();
		return;
	}
	if *week == 0 {
		fmt.Println("ERROR: No week specified");
		flag.PrintDefaults();
		return;
	}

	// Beginning of the JSON data structure creation with 
	// basic information about this particular menu
	//	
	jsonData := new(DataStruct);
	jsonData.City = *city;
	jsonData.Week = *week;
	
	fmt.Printf("Downloading information for %s and week %i\n", *city, *week);

	// Iterates all weekdays
	//
	for day := 0; day < 5; day++ {
	
		// Downloads the current menu from the web
		// NOTE: week variable in URL must be provided from the input
		//
		res, _, err = http.Get(fmt.Sprintf("%s&veckodag=%s", *url, weekdays[day]));
	
		// No error, continue reading HTML data into the inData variable
		//
		if err == nil {
			fmt.Printf("OK, response is %s length is %i\n", res.Status, res.ContentLength);
			inData, err = ioutil.ReadAll(res.Body);
			res.Body.Close();
		} else {
			log.Println(err);
		}
						
		// No error reading data from the HTML document, continue building the 
		// JSON data structure with current day and parse the HTML data
		// 
		if err == nil {
			jsonData.Days[day].Day 		= day;
			jsonData.Days[day].Name 	= weekdays[day];
			jsonData.Days[day].Restaurants 	= Parse(inData);
		} else {
			log.Println(err);
		}
	}

	// Generate the JSON code from the data structure
	//
	var output        = bytes.NewBuffer(make([]byte, 0));
	var jsonOutput, _ = json.Marshal(jsonData);
	json.Indent(output, jsonOutput, "", "");		// Indent with no prefix
	
	// Convert the JSON data from *Buffer to a []byte slice 
	//
	var outData = make([]byte, output.Len());
	var n,_ = output.Read(outData);

	// Compute the md5 hash value of the JSON data
	//
	var hashStr, hash = GenerateHash(outData);
	fmt.Printf("MD5 is: %s\n", hashStr);
	
	// Write JSON data to output file
	//
	fmt.Printf("Writing %i bytes to %s\n", n, *out);
	err = ioutil.WriteFile(*out, outData, 0644);
	
	if err != nil {
		log.Println(err);
	}
	
	// Write MD5 hash to file
	//
	fmt.Printf("Writing md5 sum\n");
	err = ioutil.WriteFile(fmt.Sprintf("%s.md5", *out), hash, 0644);
	if err != nil {
		log.Println(err);
	}
}

// Function for parsing out the real information from the HTML document
//
func Parse(in []byte) ([]RestData) {
	// Setup a string to work with and some regular expressions
	// 
	var strData  = string(in);
	var rx_image = regexp.MustCompile("SRC=\"([^\"]+)\"");
	var rx_text  = regexp.MustCompile("<center>(.+)</center>");
	var rx_html  = regexp.MustCompile("<[^>]+>");
	
	// The HTML document is split into sections of one resturant per
	// item in the tds slice. 
	//
	var tds = strings.Split(strData, "<TD WIDTH=\"130\" ALIGN=\"CENTER\" VALIGN=\"TOP\" BGCOLOR=\"#FFFFFF\">", -1);

	// Create RestData slice to add restaurant data to, length is computed
	// from the number of items in the tds slice.
	// NOTE: Minus one is needed because tds[0] represent the data _before_
	//	 The first restaurant entry appears.
	//
	var restaurant = make([]RestData, (len(tds) - 1));

	// Iterate all restaurants from the HTML document
	//
	for i := 1; i < len(tds); i++ {

		// Various regular expressions and splits in order to parse out
		// the important information from all the junky HTML parts.
		//
		var image 	= rx_image.FindAllStringSubmatch(tds[i], -1);
		var name	= MatchRestaurant(image[0][1]);
		var tmpText 	= rx_text.FindAllString(tds[i], -1);
		var tmpMenu0  	= strings.Split(tds[i], "<TD WIDTH=\"311\" VALIGN=\"TOP\" BGCOLOR=\"#FFFFFF\"><IMG SRC=\"../grafik/space.gif\" BORDER=0 width=\"1\" HEIGHT=\"5\">", -1);
		var tmpMenu1 	= strings.Split(tmpMenu0[1], "</TD>", -1);
		var menu 	= strings.Replace(tmpMenu1[0], "<LI>", "* ", -1);
				
		// Replace all <br> and <br/> tags with newlines instead (\n)
		//
		menu = strings.Replace(menu, "<BR>", "\n", -1);
		menu = strings.Replace(menu, "<br/>", "\n", -1);
		
		// Compute real index and add restaurant data to the RestData slice
		// 
		var index = i - 1;
	
		restaurant[index].ImageUrl 	= fmt.Sprintf("http://service.dt.se/lunch/%s", image[1]);
		restaurant[index].Name 		= name;
		restaurant[index].Menu 		= strings.TrimSpace(menu);
		
		// If a "subtext" or description is found (the short text beneath
		// the image in the menu). Parse out all HTML from it and save it 
		// to the RestData slice
		//
		if len(tmpText) > 0 {
			var text = rx_html.ReplaceAllString(tmpText[0], " ");
			restaurant[index].Description = text;
		} else {
			restaurant[index].Description = "";
		}
	}
	return restaurant;
}

// Function for trying to determine the name of the current restaurants
// (Since that information isn't avalible on the web, only in the images)
//
func MatchRestaurant(in string) string {

	// Very large two dimensional string slice for matching the 
	// image with the real name of the restaurant
	// TODO: Implement with a HashTable 
	//
	arr := [][]string { 
		// Falun
		[]string { "lunchlogo/club-etage.gif", 			"Club Etage" },
		[]string { "lunchlogo/chinathai.gif", 			"Restaurang China Thai" },
		[]string { "lunchlogo/hemkop.gif", 			"Hemk&ouml;p" },
		[]string { "lunchlogo/LugnetMatEvent.gif", 		"Lugnet Mat &amp; Event" },
		[]string { "lunchlogo/Z-KROG.gif",			"Z-krog" },
		[]string { "lunchlogo/City_Life.gif",			"City Life" },
		[]string { "lunchlogo/geschwornergarden_09.gif",	"Geschwornerg&auml;rden" },
		[]string { "lunchlogo/Gamla-staberg-2010.gif",		"Gamla Staberg" },
		[]string { "lunchlogo/koppis.gif", 			"Restaurang Koppis" },
		[]string { "lunchlogo/carianna.gif",			"Restaurang Cari Anna" },
		[]string { "lunchlogo/marianns_05.gif",			"Mariann's Saloon" },
		[]string { "lunchlogo/dalasalen_dalreg.gif",		"Dalasalen" },
		[]string { "lunchlogo/kuselska-rappans.gif",		"K&uuml;selska Krogen" },
		[]string { "lunchlogo/framby-udde-2.gif",		"Runns aktivitetscenter" },
		[]string { "lunchlogo/hammars.gif",			"Hammars" },
		[]string { "lunchlogo/Restaurang_Chapeau_dor.gif",	"Chapeau d'or" },
		[]string { "lunchlogo/ah.gif",				"&Aring;h" },
		[]string { "lunchlogo/haganas.gif",			"Hagan&auml;s" },
		[]string { "lunchlogo/Pitchers.gif",			"Pitchers" },
		[]string { "lunchlogo/Dossbergets-vardshus.gif",	"D&ouml;ssbergets v&auml;rdshus" },
		[]string { "lunchlogo/trotzgatan3.gif",			"Trotzgatan 3" }, 
		[]string { "lunchlogo/Victuscella.gif",			"Victuscella" },
		[]string { "lunchlogo/Scandic_lugnet.gif",		"Scandic" },
		[]string { "lunchlogo/HettoVilt.gif",			"Hett &amp; Vilt" },
		[]string { "lunchlogo/Yrkesakademin.gif",		"Yrkesakademin" },

		// Borlange
		[]string { "lunchlogo/BlgHV.gif",			"Borl&auml;nge Hotel &amp; V&auml;rdshus" },
		[]string { "lunchlogo/liljan.gif",			"Restaurang Liljan" },
		[]string { "lunchlogo/Tzatziki-blge.gif",		"Tzatziki" },
		[]string { "lunchlogo/thai-o-sushi.gif",		"Restaurang Thai &amp; Sushi" },
		[]string { "lunchlogo/Dalaflyget.gif",			"Dalaflyget" },
		[]string { "lunchlogo/subway.gif",			"Subway" },
		[]string { "lunchlogo/buskakersgastgiv.gif",		"Busk&aring;kers G&auml;stgifvarg&aring;rd" },
		[]string { "lunchlogo/matpalatset.gif",			"Matpalatset" },
		[]string { "lunchlogo/octaven_logo.gif",		"Restaurang Octaven" },
		[]string { "lunchlogo/bla_lagan.gif",			"Bl&aring; L&aring;gan" },
		[]string { "lunchlogo/coop_forum.gif",			"Coop Forum" },
		[]string { "lunchlogo/Festmakarna06.gif",		"Festmakarna" },
		[]string { "lunchlogo/kok-nystrom.gif",			"K&ouml;k Nystr&ouml;m restaurang &amp; catering" },
		[]string { "lunchlogo/Lilla-Krogen_2010.gif",		"Gamla Lilla Krogen Werners" },
		[]string { "lunchlogo/matopotatis.gif",			"Mat &amp; Potatis" },
		[]string { "lunchlogo/Officerssalongen-2010.gif",	"Officiersalongen" },
		[]string { "lunchlogo/Restaurang-Fortuna-09.gif",	"Restaurang Fortuna" },
		[]string { "lunchlogo/Sushilovers.gif",			"Sushi Lovers" },
		[]string { "lunchlogo/travinn.gif",			"Trav Inn" },
		[]string { "lunchlogo/ya.gif",				"Yrkesakademin" },
		[]string { "lunchlogo/Scandic_blge.gif",		"Scandic" },
		[]string { "lunchlogo/TeknikdRest.gif",			"Teknikdalens Restaurang" },
		[]string { "lunchlogo/Broken-Dreams-borlange.gif",	"Broken Dreams" },
		[]string { "lunchlogo/Wild_West_Restaurang.gif",	"Wild West Restaurang" },
		[]string { "lunchlogo/The-Rock-House.gif",		"The Rock House" },
		[]string { "lunchlogo/Mathornan-Galaxen.gif",		"Math&ouml;rnan Galaxen" },
		[]string { "lunchlogo/bragematsalen.gif",		"Brage Matsalen" },
		[]string { "lunchlogo/matlagarna.gif", 			"Matlagarna" },

		//Ludvika
		[]string { "lunchlogo/Ahlens_cafe.gif",			"&Aringhl&eacute;ns caf&eacute;" },
		[]string { "lunchlogo/Gallerian.gif", 			"Restaurang &amp; Cafe Gallerian" },
		[]string { "lunchlogo/Hagge_Golfkrog_20105.gif",	"Hagge Golfkrog" },
		[]string { "lunchlogo/Kan-Elen-logo.gif",		"Kan Elen" },
		[]string { "lunchlogo/Piren_2009.gif",			"Restaurang Piren" },
		[]string { "lunchlogo/pizzeria_milano.gif",		"Pizzeria Milano" },
		[]string { "lunchlogo/silverdollar.gif",		"Silverdollar" },
		[]string { "lunchlogo/smedjebackens-wardshus.gif",	"Smedjebackens W&auml;rdshus" },
		[]string { "lunchlogo/Stations_Kiosken.gif",		"Stations Kiosken" },
		[]string { "lunchlogo/stopet.gif",			"Hotell &amp; V&aumlrdshus Stopet" },
		[]string { "lunchlogo/Sussis-Mat.gif",			"Sussi's Mat &amp; Catering" },
		[]string { "lunchlogo/Wanbo-Herrgard.gif", 		"Wanbo Herrg&aring;rd" },
		[]string { "lunchlogo/Viljan-cafe.gif",			"Viljan" },
		[]string { "lunchlogo/Gourmet.gif",			"Restaurang Gourmet Pizzeria" },
		[]string { "lunchlogo/Kyrkogatan-no-9.gif",		"Kyrkogatan no. 9" },
		[]string { "lunchlogo/McDonalds2010.gif", 		"McDonalds" },

		//Mora
		[]string { "lunchlogo/Backa-Herrgard_09.gif",		"B&auml;cka Herrg&aring;rd" },
		[]string { "lunchlogo/bykrogen2.gif",			"Bykrogen" },
		[]string { "lunchlogo/Cafe_Oscar.gif",			"Restaurang &amp; Caf&eacute; Oscar" },
		[]string { "lunchlogo/Hotell-Alvdalen.gif",		"Hotell &Auml;lvdalen" },
		[]string { "lunchlogo/hotell-kung-gosta.gif",		"Hotell Kung G&ouml;sta" },
		[]string { "lunchlogo/moraparken.gif", 			"Mora Parken" },
		[]string { "lunchlogo/Orsa_Stadshotell.gif",		"Orsa Stadshotell" },
		[]string { "lunchlogo/Strand-kok-o-bar.gif",		"strand K&ouml;k &amp; Bar" },
		[]string { "lunchlogo/Vasagatan-32.gif",		"Restaurang Vasagatan 32" },
		[]string { "lunchlogo/Wasastugan.gif",			"Restaurang Wasastugan" },
		[]string { "lunchlogo/vi_pa_hornet.gif",		"Vi p&aring; H&ouml;rnet" },
		[]string { "lunchlogo/Orsa-Stadshotell.gif",		"Orsa Stadshotell" },
		[]string { "lunchlogo/FM-Mattson.gif",			"FM Mattsson arena" },
		[]string { "lunchlogo/Noret-Restaurang.gif",		"Noret Restaurang &amp; Pizzeria" },
		[]string { "lunchlogo/Pasha-restaurang2010.gif",	"Pasha Restaurang &amp; Pizzeria" },
		[]string { "lunchlogo/Famous-Moose-Restaurang.gif",	"Famous Moose" },
		[]string { "lunchlogo/Jacob.gif", 			"Jacob restaurang &amp; bar" },
		[]string { "lunchlogo/Ljungbergs-Sportsbar.gif", 	"Ljungbergs sportsbar" },
		[]string { "lunchlogo/Wibe-Restaurangen.gif", 		"Wibe Restaurangen" },

		//Sater/Hedemora
		[]string { "lunchlogo/akropolis_sdt.gif",		"Restaurang Akropolis" },
	 	[]string { "lunchlogo/bla-lagunen.gif",			"Bl&aring; Lagunen" },
	 	[]string { "lunchlogo/lappens.gif",			"Lappens V&auml;gkrog" },
	 	[]string { "lunchlogo/restaurang-skonvik.gif",		"Restaurang Sk&ouml;nvik" },
	 	[]string { "lunchlogo/The_Kings_Arms_2.gif",		"The Kings Arms" },
	 	[]string { "lunchlogo/Restaurang-Tjarna-Brunn.gif",	"Restaurang Tj&auml;rna Brunn" },
	 	[]string { "lunchlogo/tjarna-brunn.gif",		"Restaurang Tj&auml;rna Brunn" },
	 	[]string { "lunchlogo/Pizzeria-Athena.gif", 		"Pizzeria Athena" } };

	// Simple string matching
	//
	for i := 0; i < len(arr); i++ {
		if arr[i][0] == in {
			return arr[i][1];
		}
	}
	
	fmt.Printf("WARNING: Unable to match restaurant name to image %s! Code needs updating!\n", in);
	
	return "";
}

// Simple function for generating the md5 hash of the input
// data both as a string and a []byte value. 
//
func GenerateHash(inData []byte) (hashStr string, hash []byte) {

	// Create new hash object
	//
	var h = md5.New();

	// Input data and let it do it's work
	h.Write(inData);
	var sum = h.Sum();

	// Convert the result to hexadecimal notation
	//
	hashStr = fmt.Sprintf("%x", sum);

	// Make a []byte copy 
	//
	hash = make([]byte, len(hashStr));
	strings.NewReader(hashStr).Read(hash);
	
	return;
}
