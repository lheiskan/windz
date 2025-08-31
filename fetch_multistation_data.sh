#!/bin/bash

# Script to fetch multi-station wind data from FMI API
# Usage: ./fetch_multistation_data.sh [hours_back] [output_file]
#
# Examples:
#   ./fetch_multistation_data.sh                    # Last 1 hour, saves to test_three_station_response.xml
#   ./fetch_multistation_data.sh 2                  # Last 2 hours
#   ./fetch_multistation_data.sh 1 my_data.xml      # Last 1 hour, saves to my_data.xml
#   ./fetch_multistation_data.sh 24 daily_data.xml  # Last 24 hours

# Configuration
HOURS_BACK=${1:-1}  # Default to 1 hour
OUTPUT_FILE=${2:-"test_three_station_response.xml"}

# Station IDs (modify these to fetch different stations)
# Current stations:
# 101023 - Emäsalo (Porvoo)
# 100996 - Harmaja (Helsinki Maritime)
# 151028 - Vuosaari (Helsinki)
STATION_IDS=(
    "101023"
    "100996"
    "151028"
)

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
echo "FMI Multi-Station Wind Data Fetcher"
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
    echo "  - 101023: Emäsalo (Porvoo)"
    echo "  - 100996: Harmaja (Helsinki Maritime)"
    echo "  - 151028: Vuosaari (Helsinki)"
    
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