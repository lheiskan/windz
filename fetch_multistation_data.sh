#!/bin/bash

# @vibe: 🤖 fully-ai
#
# Script to fetch single or multi-station wind data from FMI API
# Usage: 
#   ./fetch_multistation_data.sh [station_id] [hours_back] [output_file]
#   ./fetch_multistation_data.sh [hours_back] [output_file]
#   ./fetch_multistation_data.sh -h | --help
#
# Examples:
#   ./fetch_multistation_data.sh                         # Last 1 hour, 3 default stations
#   ./fetch_multistation_data.sh 2                       # Last 2 hours, 3 default stations
#   ./fetch_multistation_data.sh 100996                  # Single station (Harmaja), 1 hour
#   ./fetch_multistation_data.sh 100996 2                # Single station, 2 hours
#   ./fetch_multistation_data.sh 100996 2 harmaja.xml    # Single station, 2 hours, custom file
#   ./fetch_multistation_data.sh 24 daily.xml            # 3 default stations, 24 hours

# Show help if requested
if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    echo "FMI Wind Data Fetcher - Fetch wind data from Finnish Meteorological Institute"
    echo ""
    echo "Usage:"
    echo "  ./fetch_multistation_data.sh                      # Default: 3 stations, 1 hour"
    echo "  ./fetch_multistation_data.sh [hours]              # 3 stations, custom hours"
    echo "  ./fetch_multistation_data.sh [station_id]         # Single station, 1 hour"
    echo "  ./fetch_multistation_data.sh [station_id] [hours] # Single station, custom hours"
    echo ""
    echo "Available station IDs:"
    echo "  Porkkala Area:"
    echo "    101023 - Emäsalo (Porvoo)"
    echo "    100996 - Harmaja (Helsinki Maritime)"
    echo "    151028 - Vuosaari (Helsinki)"
    echo "    101022 - Kalbådagrund (Porkkala)"
    echo "    105392 - Itätoukki (Sipoo)"
    echo ""
    echo "  Coastal Stations:"
    echo "    100969 - Bågaskär (Inkoo)"
    echo "    100965 - Jussarö (Raasepori)"
    echo "    100946 - Tulliniemi (Hanko)"
    echo "    100932 - Russarö (Hanko)"
    echo "    100945 - Vänö (Kemiönsaari)"
    echo "    100908 - Utö (Archipelago HELCOM)"
    echo ""
    echo "  Northern Coastal:"
    echo "    101267 - Tahkoluoto (Pori)"
    echo "    101661 - Tankar (Kokkola)"
    echo "    101673 - Ulkokalla (Kalajoki)"
    echo "    101784 - Marjaniemi (Hailuoto)"
    echo "    101794 - Vihreäsaari (Oulu)"
    echo ""
    echo "Examples:"
    echo "  ./fetch_multistation_data.sh                     # Default 3 stations, 1 hour"
    echo "  ./fetch_multistation_data.sh 24                  # Default 3 stations, 24 hours"
    echo "  ./fetch_multistation_data.sh 100996              # Harmaja station only, 1 hour"
    echo "  ./fetch_multistation_data.sh 101022 2            # Kalbådagrund, 2 hours"
    echo "  ./fetch_multistation_data.sh 100908 24 uto.xml   # Utö, 24 hours, custom file"
    exit 0
fi

# Check if first argument is a station ID (numeric)
if [[ "$1" =~ ^[0-9]{6}$ ]]; then
    # Single station mode
    SINGLE_STATION_ID=$1
    HOURS_BACK=${2:-1}
    OUTPUT_FILE=${3:-"station_${SINGLE_STATION_ID}_response.xml"}
    STATION_IDS=("$SINGLE_STATION_ID")
else
    # Multi-station mode (default 3 stations)
    HOURS_BACK=${1:-1}
    OUTPUT_FILE=${2:-"test_three_station_response.xml"}
    
    # Default stations:
    # 101023 - Emäsalo (Porvoo)
    # 100996 - Harmaja (Helsinki Maritime)
    # 151028 - Vuosaari (Helsinki)
    STATION_IDS=(
        "101023"
        "100996"
        "151028"
    )
fi

# You can add more stations by uncommenting below:
# Additional Porkkala area stations:
# STATION_IDS+=("101022")  # Kalbådagrund (Porkkala lighthouse)
# STATION_IDS+=("105392")  # Itätoukki (Sipoo)

# Additional coastal stations:
# STATION_IDS+=("100969")  # Bågaskär (Inkoo)
# STATION_IDS+=("100965")  # Jussarö (Raasepori)
# STATION_IDS+=("100946")  # Tulliniemi (Hanko)
# STATION_IDS+=("100932")  # Russarö (Hanko)
# STATION_IDS+=("100945")  # Vänö (Kemiönsaari)
# STATION_IDS+=("100908")  # Utö (Archipelago HELCOM)

# Calculate time range
END_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
START_TIME=$(date -u -v-${HOURS_BACK}H +"%Y-%m-%dT%H:%M:%SZ")

# Build the FMI API URL
BASE_URL="https://opendata.fmi.fi/wfs"
PARAMS="service=WFS&version=2.0.0&request=getFeature"
PARAMS="${PARAMS}&storedquery_id=fmi::observations::weather::multipointcoverage"
PARAMS="${PARAMS}&starttime=${START_TIME}"
PARAMS="${PARAMS}&endtime=${END_TIME}"

# Add station IDs as multiple fmisid parameters
for station in "${STATION_IDS[@]}"; do
    PARAMS="${PARAMS}&fmisid=${station}"
done

# Add wind parameters
PARAMS="${PARAMS}&parameters=windspeedms,windgust,winddirection"

# Build full URL
FULL_URL="${BASE_URL}?${PARAMS}"

# Display info
echo "=========================================="
if [ ${#STATION_IDS[@]} -eq 1 ]; then
    echo "FMI Single-Station Wind Data Fetcher"
else
    echo "FMI Multi-Station Wind Data Fetcher"
fi
echo "=========================================="
echo "Time range: ${START_TIME} to ${END_TIME}"
echo "Stations: ${STATION_IDS[@]}"
echo "Output file: ${OUTPUT_FILE}"
echo ""
echo "Fetching data from FMI API..."
echo ""

# Fetch the data
if curl -s "${FULL_URL}" -o "${OUTPUT_FILE}"; then
    # Check if we got an error response
    if grep -q "ExceptionReport" "${OUTPUT_FILE}"; then
        echo "❌ Error: FMI API returned an error response:"
        grep "ExceptionText" "${OUTPUT_FILE}" | sed 's/.*<ExceptionText>//;s/<\/ExceptionText>.*//'
        exit 1
    fi
    
    # Get file size and validate
    FILE_SIZE=$(wc -c < "${OUTPUT_FILE}")
    if [ ${FILE_SIZE} -lt 1000 ]; then
        echo "❌ Error: Response seems too small (${FILE_SIZE} bytes)"
        echo "Response content:"
        cat "${OUTPUT_FILE}"
        exit 1
    fi
    
    # Count stations in response
    STATION_COUNT=$(grep -c "<target:Location" "${OUTPUT_FILE}" 2>/dev/null || echo "0")
    
    # Count observations (position entries)
    OBS_COUNT=$(grep -o "[0-9]\+\.[0-9]\+ [0-9]\+\.[0-9]\+ [0-9]\+" "${OUTPUT_FILE}" | wc -l | tr -d ' ')
    
    echo "✅ Success! Data saved to ${OUTPUT_FILE}"
    echo ""
    echo "Summary:"
    echo "  - File size: $(echo "scale=1; ${FILE_SIZE}/1024" | bc) KB"
    echo "  - Stations found: ${STATION_COUNT}"
    echo "  - Total observations: ${OBS_COUNT}"
    echo "  - Observations per station: $((OBS_COUNT / STATION_COUNT))"
    echo ""
    echo "Station details:"
    for station in "${STATION_IDS[@]}"; do
        case "$station" in
            "101023") echo "  - ${station}: Emäsalo (Porvoo)" ;;
            "100996") echo "  - ${station}: Harmaja (Helsinki Maritime)" ;;
            "151028") echo "  - ${station}: Vuosaari (Helsinki)" ;;
            "101022") echo "  - ${station}: Kalbådagrund (Porkkala)" ;;
            "105392") echo "  - ${station}: Itätoukki (Sipoo)" ;;
            "100969") echo "  - ${station}: Bågaskär (Inkoo)" ;;
            "100965") echo "  - ${station}: Jussarö (Raasepori)" ;;
            "100946") echo "  - ${station}: Tulliniemi (Hanko)" ;;
            "100932") echo "  - ${station}: Russarö (Hanko)" ;;
            "100945") echo "  - ${station}: Vänö (Kemiönsaari)" ;;
            "100908") echo "  - ${station}: Utö (Archipelago HELCOM)" ;;
            "101267") echo "  - ${station}: Tahkoluoto (Pori)" ;;
            "101661") echo "  - ${station}: Tankar (Kokkola)" ;;
            "101673") echo "  - ${station}: Ulkokalla (Kalajoki)" ;;
            "101784") echo "  - ${station}: Marjaniemi (Hailuoto)" ;;
            "101794") echo "  - ${station}: Vihreäsaari (Oulu)" ;;
            *) echo "  - ${station}: (Unknown station)" ;;
        esac
    done
    
else
    echo "❌ Error: Failed to fetch data from FMI API"
    exit 1
fi

echo ""
echo "To view the raw XML:"
echo "  less ${OUTPUT_FILE}"
echo ""
echo "To test with the FMI parser:"
echo "  go test -v ./pkg/fmi -run TestParseMultiStationResponse"
echo ""
echo "URL used:"
echo "${FULL_URL}"
