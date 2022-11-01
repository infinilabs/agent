decrypt_auth() {
    local func_name="decrypt_auth"
    local enc_base64="${1}"
    # echo -n "xxxxxxx" | od -A n -t x1 | tr -d ' '
    local enc_key="${2}"
    local enc_iv="${3}"
    local enc_type="${4}"

    printf "%s\n" "${enc_base64}" | openssl enc -d "${enc_type}" -base64 -K "${enc_key}" -iv "${enc_iv}" 2>/dev/null || {
        error "${func_name}" "openssl enc failed"
        return 2
    }
}

decrypt_auth "${1}" "${2}" "${3}" "${4}"