import { ArrowLeft, ArrowRight, MapPin } from "lucide-react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useEffect, useState } from "react";

import ProductCard from "../../components/ProductCard";
import ErrorCard from "../../components/ErrorCard";
import SearchHeader from "../../components/SearchHeader";
import PageLoader from "../../components/PageLoader";

import useAxios from "../../hooks/useAxios";
import useTitle from "../../hooks/useTitle";
import { useUserState } from "../../store/user";
import {
  ProductResponse,
  Response,
} from "../../types/response";
import {
  API_GENERAL_PRODUCTS,
  API_METHOD_GET,
  API_USER_PRODUCTS,
  PATH_ADDRESS,
  PATH_LOGIN,
  PRODUCTS_TITLE,
  TOKEN_KEY,
} from "../../const/const";
import Button from "../../components/Button";

export default function Products() {
  const limitPerPage = 10;
  const [page, setPage] = useState(1);
  const [totalPage, setTotalPage] = useState(1);

  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  useTitle(PRODUCTS_TITLE);

  const userState = useUserState();

  const search = searchParams.get("search");
  const categoryDecode = searchParams.get("category")?.split("-");
  const categoryId = categoryDecode ? parseInt(categoryDecode[0]) : 0;
  const category = categoryDecode
    ? categoryDecode
        .filter(function (_, i) {
          return 0 !== i;
        })
        .join("-")
    : "";

  let apiPath = API_GENERAL_PRODUCTS;
  if (userState.id !== 0) {
    apiPath = API_USER_PRODUCTS;
  }

  const {
    data: products,
    isLoading,
    error,
  } = useAxios<Response<ProductResponse>>(
    apiPath +
      "?" +
      `page=${page}` +
      `&limit=${limitPerPage}` +
      (search ? `&name=${search}` : "") +
      (categoryId ? `&category_id=${categoryId}` : ""),
    API_METHOD_GET
  );

  useEffect(() => {
    const currPage =
      products && products.data && products.data.pagination
        ? products.data.pagination.current_page
        : 1;
    const totalPage =
    products && products.data && products.data.pagination
        ? products.data.pagination.total_page
        : 1;

    setTotalPage(totalPage);
    setPage(currPage);
  }, [products]);

  return (
    <>
      <SearchHeader />
      <div className="grow flex bg-primary-white justify-center">
        <div className="my-16 w-[90%] max-w-[1259px] flex flex-col items-center gap-10">
          <div className="w-full flex flex-col justify-between gap-4">
            <div className="flex flex-col gap-4">
              <h1 className="text-2xl font-bold text-primary-black">
                {search
                  ? `Search for: "${search}"`
                  : category
                  ? category
                  : "All Products"}
              </h1>
              <p className="text-primary-black font-bold">
                Showing results for:{" "}
                <span
                  className="cursor-pointer text-primary-gray font-normal"
                  onClick={() => {
                    if (localStorage.getItem(TOKEN_KEY)) {
                      navigate(PATH_ADDRESS);
                    } else {
                      navigate(PATH_LOGIN);
                    }
                  }}
                >
                  Your location <MapPin className="inline" />
                </span>
              </p>
            </div>
          </div>
          {error ? (
            <ErrorCard errors={error} />
          ) : isLoading ? (
            <PageLoader />
          ) : products?.data.products.length === 0 ? (
            <div className="w-full grow flex items-center justify-center">
              <div className="flex flex-col gap-2.5">
                <p className="text-primary-black font-semibold text-2xl">
                  No Results
                </p>
                <p className="text-primary-black">
                  Do you want to browse other products or{" "}
                  <span
                    className="cursor-pointer text-primary-blue font-bold"
                    onClick={() => {
                      if (localStorage.getItem(TOKEN_KEY)) {
                        navigate(PATH_ADDRESS);
                      } else {
                        navigate(PATH_LOGIN);
                      }
                    }}
                  >
                    change location?
                  </span>
                </p>
              </div>
            </div>
          ) : (
            <>
             <div className="w-full grid grid-cols-2 lg:grid-cols-5 justify-between gap-y-6 gap-x-2 md:justify-center md:gap-5">
          {products &&
            products.data &&
            products.data.products &&
            products.data.products.map((item) => {
              return <ProductCard key={item.id} product={item} />;
            })}
        </div>
        <div className="flex gap-2 mt-3">
          <div className="w-[50px]">
            <Button
              size="sm"
              square={true}
              onClick={() => {
                setPage((page) => (page > 1 ? page - 1 : page));
              }}
              disabled={page <= 1}
            >
              <div className="flex justify-center items-center">
                <ArrowLeft />
              </div>
            </Button>
          </div>
          <div className="w-[50px]">
            <Button
              size="sm"
              square={true}
              onClick={() => {
                setPage((page) =>
                  page < totalPage ? page + 1 : page
                );
              }}
              disabled={page >= totalPage}
            >
              <div className="flex justify-center items-center">
                <ArrowRight />
              </div>
            </Button>
          </div>
        </div>
            </>
          )}
        </div>
      </div>
    </>
  );
}
